package worker

import (
	"context"
	"log"
	"time"

	"cassiopeia/internal/service"
)

type NASAWorker struct {
	service  service.NASAService
	interval time.Duration
	stopChan chan struct{}
	running  bool
}

func NewNASAWorker(service service.NASAService, interval time.Duration) *NASAWorker {
	return &NASAWorker{
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (w *NASAWorker) Start() {
	if w.running {
		return
	}

	w.running = true
	log.Printf("NASA Worker started with interval %v", w.interval)

	// Запускаем сразу первую синхронизацию
	w.syncNASA()

	// Затем запускаем периодическую
	go w.run()
}

func (w *NASAWorker) Stop() {
	if !w.running {
		return
	}

	close(w.stopChan)
	w.running = false
	log.Println("NASA Worker stopped")
}

func (w *NASAWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.syncNASA()
		case <-w.stopChan:
			return
		}
	}
}

func (w *NASAWorker) syncNASA() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Println("NASA Worker: Starting sync...")

	// 1. Синхронизируем OSDR данные
	if err := w.service.FetchAndStoreOSDR(ctx); err != nil {
		log.Printf("NASA Worker OSDR error: %v", err)
	} else {
		log.Println("NASA Worker: OSDR data synced")
	}

	// 2. Получаем APOD
	if err := w.service.FetchAndStoreAPOD(ctx); err != nil {
		log.Printf("NASA Worker APOD error: %v", err)
	} else {
		log.Println("NASA Worker: APOD data updated")
	}

	// 3. Получаем NEO данные
	if err := w.service.FetchAndStoreNEO(ctx); err != nil {
		log.Printf("NASA Worker NEO error: %v", err)
	} else {
		log.Println("NASA Worker: NEO data updated")
	}

	log.Println("NASA Worker: Sync completed")
}
