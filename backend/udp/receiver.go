package udp

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"time"

	"ballistics-system/models"
)

type Receiver struct {
	port     int
	conn     *net.UDPConn
	dataChan chan<- *models.SensorData
}

func NewReceiver(port int, dataChan chan<- *models.SensorData) *Receiver {
	return &Receiver{
		port:     port,
		dataChan: dataChan,
	}
}

func (r *Receiver) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(r.port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	r.conn = conn
	log.Printf("UDP receiver listening on port %d", r.port)

	go r.receiveLoop()
	return nil
}

func (r *Receiver) receiveLoop() {
	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP read error from %v: %v", remoteAddr, err)
			continue
		}

		var data models.SensorData
		if err := json.Unmarshal(buf[:n], &data); err != nil {
			log.Printf("JSON parse error: %v, raw: %s", err, string(buf[:n]))
			continue
		}

		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}

		r.dataChan <- &data
	}
}

func (r *Receiver) Stop() {
	if r.conn != nil {
		r.conn.Close()
	}
}
