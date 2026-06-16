package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"ballistics-system/models"
)

type AlertPusher struct {
	client   mqtt.Client
	topic    string
	broker   string
	alertChan chan *models.Alert
}

func NewAlertPusher(broker, clientID, topic, username, password string) *AlertPusher {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetCleanSession(true)

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT connected to", broker)
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	}

	pusher := &AlertPusher{
		broker:    broker,
		topic:     topic,
		alertChan: make(chan *models.Alert, 100),
	}
	pusher.client = mqtt.NewClient(opts)

	go pusher.connectLoop()
	go pusher.publishLoop()

	return pusher
}

func (p *AlertPusher) connectLoop() {
	for {
		if token := p.client.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("MQTT connect error: %v, retrying in 5s", token.Error())
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
}

func (p *AlertPusher) publishLoop() {
	for alert := range p.alertChan {
		p.publishAlert(alert)
	}
}

func (p *AlertPusher) publishAlert(alert *models.Alert) {
	if !p.client.IsConnected() {
		log.Println("MQTT not connected, alert queued")
		return
	}

	payload, err := json.Marshal(alert)
	if err != nil {
		log.Printf("Alert JSON marshal error: %v", err)
		return
	}

	topic := fmt.Sprintf("%s/%s/%s", p.topic, alert.AlertType, alert.DeviceID)
	token := p.client.Publish(topic, 1, false, payload)
	go func() {
		token.Wait()
		if token.Error() != nil {
			log.Printf("MQTT publish error: %v", token.Error())
		} else {
			log.Printf("Alert published: %s - %s", alert.AlertType, alert.Message)
		}
	}()
}

func (p *AlertPusher) Push(alert *models.Alert) {
	select {
	case p.alertChan <- alert:
	default:
		log.Println("Alert channel full, dropping alert")
	}
}

func (p *AlertPusher) Stop() {
	if p.client.IsConnected() {
		p.client.Disconnect(250)
	}
	close(p.alertChan)
}

type AlertChecker struct {
	deformationMax float64
	minRange       float64
	alertChan      chan<- *models.Alert
}

func NewAlertChecker(deformationMax, minRange float64, alertChan chan<- *models.Alert) *AlertChecker {
	return &AlertChecker{
		deformationMax: deformationMax,
		minRange:       minRange,
		alertChan:      alertChan,
	}
}

func (c *AlertChecker) CheckSensor(data *models.SensorData) []*models.Alert {
	var alerts []*models.Alert

	if data.ArmDeformation > c.deformationMax {
		level := "warning"
		if data.ArmDeformation > c.deformationMax*1.2 {
			level = "critical"
		}
		alert := &models.Alert{
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			AlertType:   "arm_crack_risk",
			AlertLevel:  level,
			Message:     fmt.Sprintf("弩臂变形 %.2f mm 超过阈值 %.2f mm，存在裂纹风险", data.ArmDeformation, c.deformationMax),
			SensorValue: data.ArmDeformation,
			Threshold:   c.deformationMax,
		}
		alerts = append(alerts, alert)
	}

	if alerts != nil {
		for _, a := range alerts {
			select {
			case c.alertChan <- a:
			default:
			}
		}
	}

	return alerts
}

func (c *AlertChecker) CheckRange(deviceID string, actualRange float64) *models.Alert {
	if actualRange < c.minRange {
		level := "warning"
		if actualRange < c.minRange*0.7 {
			level = "critical"
		}
		alert := &models.Alert{
			DeviceID:    deviceID,
			Timestamp:   time.Now(),
			AlertType:   "insufficient_range",
			AlertLevel:  level,
			Message:     fmt.Sprintf("射程 %.2f m 低于最低要求 %.2f m", actualRange, c.minRange),
			SensorValue: actualRange,
			Threshold:   c.minRange,
		}
		select {
		case c.alertChan <- alert:
		default:
		}
		return alert
	}
	return nil
}
