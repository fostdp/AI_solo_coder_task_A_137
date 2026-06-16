package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	UDPPort         int
	HTTPAddr        string
	ClickHouseDSN   string
	MQTTBroker      string
	MQTTClientID    string
	MQTTTopic       string
	MQTTUsername    string
	MQTTPassword    string
	DeformationMax  float64
	MinRange        float64
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		UDPPort:        getEnvInt("UDP_PORT", 8080),
		HTTPAddr:       getEnvStr("HTTP_ADDR", ":8081"),
		ClickHouseDSN:  getEnvStr("CLICKHOUSE_DSN", "clickhouse://localhost:9000?database=ballistics&username=default&password="),
		MQTTBroker:     getEnvStr("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID:   getEnvStr("MQTT_CLIENT_ID", "ballistics-alert"),
		MQTTTopic:      getEnvStr("MQTT_TOPIC", "ballistics/alerts"),
		MQTTUsername:   getEnvStr("MQTT_USERNAME", ""),
		MQTTPassword:   getEnvStr("MQTT_PASSWORD", ""),
		DeformationMax: getEnvFloat("DEFORMATION_MAX", 15.0),
		MinRange:       getEnvFloat("MIN_RANGE", 300.0),
	}
	log.Printf("Config loaded: UDP=%d HTTP=%s", cfg.UDPPort, cfg.HTTPAddr)
	return cfg
}

func getEnvStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
