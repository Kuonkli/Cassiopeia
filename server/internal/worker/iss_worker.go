package worker

import (
	"context"
	"log"
	"time"

	"cassiopeia/internal/service"
)

type ISSWorker struct {
	service   service.ISSService
	interval  time.Duration
	stopChan  chan struct{}
	isRunning bool
}

func NewISSWorker(service service.ISSService, interval time.Duration) *ISSWorker {
	return &ISSWorker{
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (w *ISSWorker) Start() {
	if w.isRunning {
		return
	}

	w.isRunning = true
	log.Printf("ISS Worker started with interval %v", w.interval)

	go w.run()
}

func (w *ISSWorker) Stop() {
	if !w.isRunning {
		return
	}

	close(w.stopChan)
	w.isRunning = false
	log.Println("ISS Worker stopped")
}

func (w *ISSWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Первый запуск сразу
	w.fetchISSData()

	for {
		select {
		case <-ticker.C:
			w.fetchISSData()
		case <-w.stopChan:
			return
		}
	}
}

func (w *ISSWorker) fetchISSData() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.service.FetchAndStoreISSData(ctx); err != nil {
		log.Printf("ISS Worker error: %v", err)
	} else {
		log.Println("ISS Worker: data fetched successfully")
	}
}
