// Package worker provides APIs for executing tasks in the system.
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Executer is a function that executes the task.
type Executer func(ctx context.Context)

// Worker manages the execution of tasks.
type Worker struct {
	wg        sync.WaitGroup
	mu        sync.RWMutex
	semaphore chan struct{}
	shutdown  chan struct{}
	running   map[string]context.CancelFunc
}

// New creates a worker with max number of goroutines that can run at aby given time.
func New(maxRunningTasks int) (*Worker, error) {
	if maxRunningTasks <= 0 {
		return nil, errors.New("max running tasks must be greater than 0")
	}

	semaphore := make(chan struct{}, maxRunningTasks)

	//fill it
	for range maxRunningTasks {
		semaphore <- struct{}{}
	}

	w := Worker{
		semaphore: semaphore,
		shutdown:  make(chan struct{}),
		running:   make(map[string]context.CancelFunc),
	}
	return &w, nil
}

// Running returns the number of running tasks.
func (w *Worker) Running() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.running)
}

// Shutdown waits for all tasks to finish before return.
func (w *Worker) Shutdown(ctx context.Context) error {
	//signal we are shutting down
	close(w.shutdown)

	//cancel all running tasks
	func() {
		//need to have lock till end of operation
		w.mu.RLock()
		defer w.mu.RUnlock()
		for _, cancel := range w.running {
			cancel()
		}
	}()

	done := make(chan struct{}, 1)

	// a goroutine responsible for waiting till all tasks terminate
	go func() {
		w.wg.Wait()
		close(done)
	}()

	//block here on ctx and done
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		//not enough time for clean shutdown
		return ctx.Err()
	}
}

// Start launches a goroutine to execute the task and returns the id for that goroutine.
func (w *Worker) Start(ctx context.Context, executer Executer) (string, error) {

	//block here waiting for a semifor, or ctx timeout or shutdown even before start.

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-w.shutdown:
		return "", errors.New("shutdown signal received")
	case <-w.semaphore:
		//move on
	}

	workerId := uuid.NewString()

	//check the ctx to be of type deadline, if not make it one with a default deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Minute)
	}

	//create a new ctx that worker controls the goroutine with this one, seperated from caller's ctx.
	ctx, cancel := context.WithDeadline(context.Background(), deadline)

	//register it into running map
	func() {
		//need to keep the lock till end, so use a literal fn
		w.mu.Lock()
		defer w.mu.Unlock()
		w.running[workerId] = cancel
	}()

	//launch a goroutine to execute the task.
	w.wg.Add(1)
	go func() {

		//separate defer for this, in case of panic in other one, allows other goroutines to execute tasks
		defer func() {
			w.semaphore <- struct{}{}
		}()

		//clean up
		defer func() {
			//call cancel even if you remove it from running
			cancel()

			func() {
				//need lock till end so we use this literal.
				w.mu.Lock()
				defer w.mu.Unlock()
				delete(w.running, workerId)
			}()

			w.wg.Done()
		}()

		//execute the task
		executer(ctx)
	}()

	return workerId, nil
}

// Stop stops the currently running task.
func (w *Worker) Stop(workerId string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if cancel, ok := w.running[workerId]; !ok {
		return fmt.Errorf("worker with id %q not found", workerId)
	} else {
		cancel()
		return nil
	}
}
