// dbtest provides with helpers to setup database for testing.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/hamidoujand/task-scheduler/business/database/postgres"
	"github.com/hamidoujand/task-scheduler/foundation/docker"
)

// NewDatabaseClient creates a container off of a postgres image and create a client to be used in testing.
func NewDatabaseClient(t *testing.T) *postgres.Client {
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

	//connect to db as main user
	masterClient, err := postgres.NewClient(postgres.Config{
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
	client, err := postgres.NewClient(postgres.Config{
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
		// close master conn
		t.Logf("deleting schema %s", schemaName)
		if _, err := masterClient.DB.ExecContext(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE"); err != nil {
			t.Fatalf("failed to delete schema %s: %s", schemaName, err)
		}
		//close both clients
		_ = masterClient.DB.Close()
		_ = client.DB.Close()
		//clean up container as well
		if err := c.Stop(); err != nil {
			t.Logf("failed to stop container %s: %s", c.Id, err)
		}
		t.Logf("removed the container successfully %s", c.Id)
	})
	return client
}
