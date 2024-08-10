package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hamidoujand/task-scheduler/business/worker"
)

func TestWorker(t *testing.T) {
	task := func(ctx context.Context) {
		t.Logf("task running")
		<-ctx.Done()
		t.Logf("task terminating")
	}

	worker, err := worker.New(4)
	if err != nil {
		t.Fatalf("expected to created a worker: %s", err)
	}

	for range 4 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		if _, err := worker.Start(ctx, task); err != nil {
			t.Fatalf("expected to execute the task: %s", err)
		}
	}

	for range 4 {
		if worker.Running() == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond) //not a falky test, since our ctx also has the same timeout
	}

	current := worker.Running()
	if current != 0 {
		t.Errorf("running= %d, got %d", 0, current)
	}

	//clean shutdown
	if err := worker.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected a clean shutdown: %s", err)
	}

}

func TestCancelWorker(t *testing.T) {
	var wg sync.WaitGroup //to make sure all tasks started before cancel
	wg.Add(4)

	//task to cancel
	task := func(ctx context.Context) {
		wg.Done() //call it at the start
		t.Log("task is running")
		<-ctx.Done() //as long as ctx allows
		t.Log("task is terminating")
	}

	worker, err := worker.New(4)
	if err != nil {
		t.Fatalf("expected to create a worker: %s", err)
	}

	//start those tasks
	for range 4 {
		//give each fn a 10 seconds timeout to wait
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if _, err := worker.Start(ctx, task); err != nil {
			t.Fatalf("expected to start task: %s", err)
		}
	}

	//now wait for all of them to start
	wg.Wait()

	//only have 1 second to shutdown clean
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := worker.Shutdown(ctx); err != nil {
		t.Fatalf("expected each task to shutdown cleanly: %s", err)
	}

	//check the number of running
	running := worker.Running()
	if running != 0 {
		t.Fatalf("running=%d, got %d", 0, running)
	}
}

func TestStopWorkers(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(4)

	task := func(ctx context.Context) {
		wg.Done()

		t.Log("task running")
		<-ctx.Done()
		t.Log("task terminating")
	}

	var workerIds []string
	worker, err := worker.New(4)
	if err != nil {
		t.Fatalf("expected to create a worker: %s", err)
	}

	for range 4 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		id, err := worker.Start(ctx, task)
		if err != nil {
			t.Fatalf("expected to start task: %s", err)
		}

		workerIds = append(workerIds, id)
	}

	//wait for all of them to start
	wg.Wait()

	for _, workerId := range workerIds {
		if err := worker.Stop(workerId); err != nil {
			t.Fatalf("expected to stop worker with id %q: %s", workerId, err)
		}
	}

	//wait for all of them to finish
	for {
		if worker.Running() == 0 {
			break
		}
		//sleep
		time.Sleep(time.Millisecond * 100)
	}

	//at the end shutdown all of them
	if err := worker.Shutdown(context.Background()); err != nil {
		t.Errorf("expected to shutdown all tasks: %s", err)
	}
}
