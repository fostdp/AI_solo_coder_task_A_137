package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ch "ballistics-system/clickhouse"
	"ballistics-system/config"
	"ballistics-system/models"
	"ballistics-system/mqtt"
	"ballistics-system/penetration"
	"ballistics-system/simulation"
	"ballistics-system/udp"
	"ballistics-system/api"
)

func main() {
	cfg := config.Load()

	store, err := ch.NewStore(cfg.ClickHouseDSN)
	if err != nil {
		log.Printf("Warning: ClickHouse connection failed: %v", err)
		log.Println("Continuing without database...")
	}
	if store != nil {
		defer store.Close()
	}

	simEngine := simulation.NewEngine()
	penAnalyzer := penetration.NewAnalyzer()

	sensorDataChan := make(chan *models.SensorData, 1000)
	alertChan := make(chan *models.Alert, 100)

	alertPusher := mqtt.NewAlertPusher(
		cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTTopic,
		cfg.MQTTUsername, cfg.MQTTPassword,
	)
	defer alertPusher.Stop()

	alertChecker := mqtt.NewAlertChecker(cfg.DeformationMax, cfg.MinRange, alertChan)

	udpReceiver := udp.NewReceiver(cfg.UDPPort, sensorDataChan)
	if err := udpReceiver.Start(); err != nil {
		log.Fatalf("Failed to start UDP receiver: %v", err)
	}
	defer udpReceiver.Stop()

	go processSensorData(sensorDataChan, store, simEngine, penAnalyzer, alertChecker, alertPusher)
	go processAlerts(alertChan, store, alertPusher)

	httpServer := api.NewServer(cfg.HTTPAddr, store, simEngine, penAnalyzer)
	go func() {
		log.Printf("HTTP server starting on %s", cfg.HTTPAddr)
		if err := httpServer.Start(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("Ballistics System started successfully")
	log.Printf("  UDP port: %d", cfg.UDPPort)
	log.Printf("  HTTP addr: %s", cfg.HTTPAddr)
	log.Printf("  MQTT broker: %s", cfg.MQTTBroker)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	time.Sleep(500 * time.Millisecond)
	log.Println("Ballistics System stopped")
}

func processSensorData(
	dataChan <-chan *models.SensorData,
	store *ch.Store,
	simEngine *simulation.Engine,
	penAnalyzer *penetration.Analyzer,
	alertChecker *mqtt.AlertChecker,
	alertPusher *mqtt.AlertPusher,
) {
	for data := range dataChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		if store != nil {
			if err := store.InsertSensorData(ctx, data); err != nil {
				log.Printf("Insert sensor data error: %v", err)
			}
		}

		alertChecker.CheckSensor(data)

		if data.ArrowInitialVelocity > 0 {
			simParams := &models.SimulationParams{
				InitialVelocity: data.ArrowInitialVelocity,
				LaunchAngle:     45.0,
				AzimuthAngle:    0.0,
				ArrowMass:       0.2,
				ArrowDiameter:   0.012,
				DragCoefficient: 0.4,
				AirDensity:      1.225,
			}
			simResult := simEngine.Simulate(simParams)
			simResult.DeviceID = data.DeviceID

			alertChecker.CheckRange(data.DeviceID, simResult.Range)

			penResult := penAnalyzer.Analyze(
				simResult.ImpactVelocity,
				0.2,
				penetration.PlateArmor,
				penetration.BodkinPoint,
				0,
			)
			simResult.ArmorType = "plate"
			simResult.PenetrationDepth = penResult.PenetrationDepth
			simResult.PenetrationSuccess = penResult.Success

			if store != nil {
				_ = store.InsertSimulationResult(ctx, simResult)
				_ = store.InsertArmorPerformance(ctx, penAnalyzer.ToArmorPerformance(penResult))
			}

			log.Printf("[%s] v0=%.1fm/s range=%.1fm impact_v=%.1fm/s KE=%.1fJ pen=%.2fmm success=%v",
				data.DeviceID, data.ArrowInitialVelocity, simResult.Range,
				simResult.ImpactVelocity, simResult.KineticEnergy,
				simResult.PenetrationDepth*1000, simResult.PenetrationSuccess)
		}

		cancel()
	}
}

func processAlerts(alertChan <-chan *models.Alert, store *ch.Store, pusher *mqtt.AlertPusher) {
	for alert := range alertChan {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if store != nil {
			_ = store.InsertAlert(ctx, alert)
		}
		pusher.Push(alert)
		log.Printf("ALERT [%s] %s: %s", alert.AlertLevel, alert.AlertType, alert.Message)
		cancel()
	}
}
