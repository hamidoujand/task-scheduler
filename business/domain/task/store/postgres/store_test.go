package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	postgresClient "github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/business/domain/task"
	postgresRepo "github.com/hamidoujand/task-scheduler/business/domain/task/store/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
)

var container docker.Container

func TestMain(m *testing.M) {
	code := m.Run()
	//clean up container
	err := container.Stop()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func setupDatabase(t *testing.T) *postgresClient.Client {
	image := "postgres:latest"
	name := "tasks"
	port := "5432"
	dockerArgs := []string{"-e", "POSTGRES_PASSWORD=password"}
	appArgs := []string{"-c", "log_statement=all"}

	//create a container
	c, err := docker.StartContainer(image, name, port, dockerArgs, appArgs)
	if err != nil {
		t.Fatalf("failed to start container with image %q: %s", image, err)
	}

	//details of container
	t.Logf("Name/ID:  %s", c.Id)
	t.Logf("Host:Port  %s", c.HostPort)
	container = c
	//connect to db as main user
	masterClient, err := postgresClient.NewClient(postgresClient.Config{
		User:       "postgres",
		Password:   "password",
		Host:       c.HostPort,
		Name:       "postgres",
		DisableTLS: true,
	})

	if err != nil {
		t.Fatalf("failed to create master db client: %s", err)
	}

	//status check
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := masterClient.StatusCheck(ctx); err != nil {
		t.Fatalf("status check failed: %s", err)
	}

	//create a random schema inside of this db
	bs := make([]byte, 8)
	if _, err := rand.Read(bs); err != nil {
		t.Fatalf("generating random schema name: %s", err)
	}
	schemaName := "a" + hex.EncodeToString(bs)

	q := "CREATE SCHEMA " + schemaName

	if _, err := masterClient.DB.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("failed to create schema %q: %s", schemaName, err)
	}

	//new client
	client, err := postgresClient.NewClient(postgresClient.Config{
		User:       "postgres",
		Password:   "password",
		Host:       c.HostPort,
		Name:       "postgres",
		Schema:     schemaName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("failed to create a client: %s", err)
	}

	if err := masterClient.StatusCheck(ctx); err != nil {
		t.Fatalf("status check failed against slave client: %s", err)
	}

	//run migrations
	t.Logf("running migration against: %q schema", schemaName)

	if err := client.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %s", err)
	}

	//register cleanup functions to run after each test.
	t.Cleanup(func() {
		t.Helper()
		// close master conn
		t.Logf("deleting schema %s", schemaName)
		if _, err := masterClient.DB.ExecContext(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE"); err != nil {
			t.Fatalf("failed to delete schema %s: %s", schemaName, err)
		}
		//close both clients
		_ = masterClient.DB.Close()
		_ = client.DB.Close()
	})
	return client
}

func Test_Task(t *testing.T) {
	client := setupDatabase(t)
	store := postgresRepo.NewRepository(client)

	//insert a new task
	id := uuid.New()
	now := time.Now()
	tt := task.Task{
		Id:          id,
		Command:     "ls",
		Args:        []string{"-1", "-a"},
		Status:      "pending",
		ScheduledAt: now.Add(time.Hour * 10),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := store.Create(context.Background(), tt)

	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	//get the task by id
	tsk, err := store.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("should return task by id %s", id)
	}

	if tsk.Command != tt.Command {
		t.Errorf("expected command to be %q, got %q", tt.Command, tsk.Command)
	}
	if len(tsk.Args) != len(tt.Args) {
		t.Errorf("expected args length to b %d but got %d", len(tt.Args), len(tsk.Args))
	}

	diffTime := tsk.ScheduledAt.Sub(now)

	if diffTime >= time.Hour*10 {
		t.Errorf("expected diff between scheduledAt and now to be less or equal to 10 but got %s", diffTime)
	}

	//update
	tsk.Result = "data"
	tsk.ErrMessage = "no-err"
	tsk.Status = "success"

	err = store.Update(context.Background(), tsk)
	if err != nil {
		t.Fatalf("should be able to update a task: %s", err)
	}
	// get the task by id
	tsk, err = store.GetById(context.Background(), id)
	if err != nil {
		t.Fatalf("should return task by id after update %s", id)
	}

	if tsk.Status != "success" {
		t.Errorf("expected the Status to be success, but got %s", tsk.Status)
	}

	if tsk.Result != "data" {
		t.Errorf("expected the Result to be data, but got %s", tsk.Result)
	}

	if tsk.ErrMessage != "no-err" {
		t.Errorf("expected the ErrMessage to be np-err, but got %s", tsk.ErrMessage)
	}

	//delete

	if err := store.Delete(context.Background(), tsk); err != nil {
		t.Fatalf("should be able to delete a task: %s", err)
	}

	_, err = store.GetById(context.Background(), id)
	if err == nil {
		t.Fatalf("should get an error when querying a deleted task")
	}
}
