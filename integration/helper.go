package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// TestDB holds the test database connection and cleanup function
type TestDB struct {
	DB       *sql.DB
	Pool     *dockertest.Pool
	Resource *dockertest.Resource
}

// SetupTestDB sets up a PostgreSQL container using dockertest and runs migrations
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Create dockertest pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	// Set max wait time for Docker operations
	pool.MaxWait = 120 * time.Second

	// Pull and run PostgreSQL container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=testuser",
			"POSTGRES_DB=testdb",
			"listen_addresses='*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	// Set container to expire after 2 minutes to avoid orphaned containers
	if err := resource.Expire(120); err != nil {
		t.Fatalf("Could not set expiration: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://testuser:secret@%s/testdb?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseURL)

	// Wait for database to be ready
	var db *sql.DB
	if err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	// Run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("Could not create migration driver: %s", err)
	}

	// Get the migrations path - go up from integration folder to root
	migrationsPath := "../migrations"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		t.Fatalf("Migrations directory not found: %s", migrationsPath)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		t.Fatalf("Could not create migrate instance: %s", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Could not run migrations: %s", err)
	}

	return &TestDB{
		DB:       db,
		Pool:     pool,
		Resource: resource,
	}
}

// Cleanup closes the database connection and purges the Docker container
func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()

	if tdb.DB != nil {
		if err := tdb.DB.Close(); err != nil {
			t.Errorf("Could not close database: %s", err)
		}
	}

	if tdb.Pool != nil && tdb.Resource != nil {
		if err := tdb.Pool.Purge(tdb.Resource); err != nil {
			t.Errorf("Could not purge resource: %s", err)
		}
	}
}

// TruncateTables truncates all tables in the test database
func (tdb *TestDB) TruncateTables(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	tables := []string{"events", "products", "users"}

	for _, table := range tables {
		_, err := tdb.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Could not truncate table %s: %s", table, err)
		}
	}
}
