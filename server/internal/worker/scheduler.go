package worker

import (
	"log"
	"sync"
	"time"
)

type Worker interface {
	Start()
	Stop()
}

type Scheduler struct {
	workers []Worker
	wg      sync.WaitGroup
	stopped bool
	mu      sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		workers: make([]Worker, 0),
	}
}

func (s *Scheduler) AddWorker(worker Worker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workers = append(s.workers, worker)
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	log.Println("Starting scheduler with", len(s.workers), "workers")

	for _, worker := range s.workers {
		s.wg.Add(1)
		go func(w Worker) {
			defer s.wg.Done()
			w.Start()
		}(worker)
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	log.Println("Stopping scheduler...")

	// Останавливаем всех воркеров
	for _, worker := range s.workers {
		worker.Stop()
	}

	// Ждем завершения
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// Таймаут на остановку
	select {
	case <-done:
		log.Println("Scheduler stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Scheduler stop timeout")
	}
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.stopped
}
