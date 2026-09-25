package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"hirescope/backend/internal/service"
)

// ReminderWorker periodically polls and delivers scheduled interview reminders.
type ReminderWorker struct {
	notificationService service.InterviewNotificationService
	interval            time.Duration
	enabled             bool
	stopChan            chan struct{}
	doneChan            chan struct{}
	mu                  sync.Mutex
	running             bool
}

// NewReminderWorker creates a new ReminderWorker instance.
func NewReminderWorker(
	notificationService service.InterviewNotificationService,
	interval time.Duration,
	enabled bool,
) *ReminderWorker {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	return &ReminderWorker{
		notificationService: notificationService,
		interval:            interval,
		enabled:             enabled,
		stopChan:            make(chan struct{}),
		doneChan:            make(chan struct{}),
	}
}

// Start launches the background worker loop.
func (w *ReminderWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	log.Printf("[ReminderWorker] Starting background worker (interval: %v, enabled: %v)", w.interval, w.enabled)

	go func() {
		defer close(w.doneChan)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		// Run an initial tick immediately on startup
		w.runOnce()

		for {
			select {
			case <-w.stopChan:
				log.Println("[ReminderWorker] Received stop signal, shutting down worker...")
				return
			case <-ticker.C:
				w.runOnce()
			}
		}
	}()
}

func (w *ReminderWorker) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	count, err := w.notificationService.ProcessDueReminders(ctx)
	if err != nil {
		log.Printf("[ReminderWorker] Error processing due reminders: %v", err)
		return
	}
	if count > 0 {
		log.Printf("[ReminderWorker] Successfully processed %d due reminder(s)", count)
	}
}

// Stop signals the worker to terminate gracefully and blocks until complete.
func (w *ReminderWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopChan)
	<-w.doneChan
	log.Println("[ReminderWorker] Background worker stopped gracefully.")
}
