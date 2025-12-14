package worker

import (
	"context"
	"log"
	"time"

	"cassiopeia/internal/service"
)

type TelemetryWorker struct {
	service  service.TelemetryService
	interval time.Duration
	stopChan chan struct{}
	running  bool
}

func NewTelemetryWorker(service service.TelemetryService, interval time.Duration) *TelemetryWorker {
	return &TelemetryWorker{
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (w *TelemetryWorker) Start() {
	if w.running {
		return
	}

	w.running = true
	log.Printf("Telemetry Worker started with interval %v", w.interval)

	// Запускаем сразу первую генерацию
	w.generateTelemetry()

	// Затем запускаем периодическую
	go w.run()
}

func (w *TelemetryWorker) Stop() {
	if !w.running {
		return
	}

	close(w.stopChan)
	w.running = false
	log.Println("Telemetry Worker stopped")
}

func (w *TelemetryWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.generateTelemetry()
		case <-w.stopChan:
			return
		}
	}
}

func (w *TelemetryWorker) generateTelemetry() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Telemetry Worker: Generating new telemetry data...")

	_, err := w.service.GenerateTelemetry(ctx)
	if err != nil {
		log.Printf("Telemetry Worker error: %v", err)
	} else {
		log.Println("Telemetry Worker: Data generated successfully")
	}
}
