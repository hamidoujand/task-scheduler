package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	//go:embed sql/*.sql
	migrationFiles embed.FS
)

// Config represents all of the mandatory configs required to stablish a connection.
type Config struct {
	User        string
	Password    string
	Host        string
	Name        string
	Schema      string
	MaxIdleConn int
	MaxOpenConn int
	MaxIdleTime time.Duration
	MaxLifeTime time.Duration
	DisableTLS  bool
}

// Client represents a postgres client and has some extended functionalities over database/sql.
type Client struct {
	DB *sql.DB
}

// NewClient initialize a client instance and returns it.
func NewClient(conf Config) (*Client, error) {
	//default
	sslMode := "required"

	if conf.DisableTLS {
		//disable in dev
		sslMode = "disable"
	}

	q := make(url.Values)

	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	//different search path
	if conf.Schema != "" {
		q.Set("search_path", conf.Schema)
	}

	uri := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(conf.User, conf.Password),
		Host:     conf.Host,
		Path:     conf.Name,
		RawQuery: q.Encode(),
	}

	db, err := sql.Open("pgx", uri.String())

	if err != nil {
		return nil, fmt.Errorf("opening connection: %w", err)
	}

	//overwrite defaults

	//default: unlimited
	db.SetMaxOpenConns(conf.MaxOpenConn)

	//default: 2
	db.SetMaxIdleConns(conf.MaxIdleConn)

	//default: unlimited
	db.SetConnMaxIdleTime(conf.MaxIdleTime)

	//default: unlimited
	db.SetConnMaxLifetime(conf.MaxLifeTime)

	return &Client{
		DB: db,
	}, nil
}

// StatusCheck checks the status of the db to
func (c *Client) StatusCheck(ctx context.Context) error {
	//check ctx to make sure its a deadline ctx
	if _, ok := ctx.Deadline(); !ok {
		//create a default timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
	}

	//with retry
	for try := 1; ; try++ {
		pingErr := c.DB.PingContext(ctx)

		if pingErr == nil {
			break
		}

		time.Sleep(time.Millisecond * time.Duration(try) * 100)

		//check the ctx to see if still not expired
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s:%w", pingErr, err)
		}
	}

	//check ctx again
	if err := ctx.Err(); err != nil {
		return err
	}

	//check the db engine

	var result bool
	const q = "SELECT TRUE"

	if err := c.DB.QueryRowContext(ctx, q).Scan(&result); err != nil {
		return fmt.Errorf("queryRowContext: %w", err)
	}

	return nil
}

// Migrate is going to do schema migration against client.
func (c *Client) Migrate() error {
	driver, err := postgres.WithInstance(c.DB, &postgres.Config{})

	if err != nil {
		return fmt.Errorf("selecting driver: %w", err)
	}

	//create a source, "sql" means prefix of the path, "sql/*.sql"
	src, err := iofs.New(migrationFiles, "sql")

	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)

	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up: %w", err)
	}
	return nil
}
