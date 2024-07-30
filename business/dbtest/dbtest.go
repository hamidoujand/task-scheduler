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
func NewDatabaseClient(t *testing.T, name string) *postgres.Client {
	image := "postgres:latest"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	if err := masterClient.StatusCheck(ctx); err != nil {
		t.Fatalf("status check failed: %s", err)
	}

	//create a random database
	bs := make([]byte, 8)
	if _, err := rand.Read(bs); err != nil {
		t.Fatalf("generating random database name: %s", err)
	}
	// dbName := "a" + hex.EncodeToString(bs)
	dbName := "a" + hex.EncodeToString(bs)
	q := "CREATE DATABASE " + dbName

	if _, err := masterClient.DB.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("failed to create database %q: %s", dbName, err)
	}

	//new client
	client, err := postgres.NewClient(postgres.Config{
		User:       "postgres",
		Password:   "password",
		Host:       c.HostPort,
		Name:       dbName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("failed to create a client: %s", err)
	}

	t.Logf("connected to the database %s", dbName)

	// status check
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	if err := client.StatusCheck(ctx); err != nil {
		t.Fatalf("status check failed against slave client: %s", err)
	}

	//run migrations
	t.Logf("running migration against: %q database", dbName)

	if err := client.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %s", err)
	}

	//register cleanup functions to run after each test.
	t.Cleanup(func() {
		// close client
		if err := client.DB.Close(); err != nil {
			t.Fatalf("failed to close client connection: %s", err)
		}

		//terminate all conns to that random database otherwise can not delete it
		const q = `
		SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1;
		`
		if _, err := masterClient.DB.ExecContext(context.Background(), q, dbName); err != nil {
			t.Fatalf("failed to remove all connections to db %q", dbName)
		}

		t.Logf("terminated all connection to db %q", dbName)

		t.Logf("deleting database %s", dbName)
		if _, err := masterClient.DB.ExecContext(context.Background(), "DROP DATABASE "+dbName); err != nil {
			t.Fatalf("failed to delete database %s: %s", dbName, err)
		}

		//close master client
		_ = masterClient.DB.Close()
		//clean up container as well
		if err := c.Stop(); err != nil {
			t.Logf("failed to stop container %s: %s", c.Id, err)
		}
		t.Logf("removed the container successfully %s", c.Id)
	})
	return client
}
