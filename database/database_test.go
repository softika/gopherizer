package database

import (
	"context"
	"log"
	"testing"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/testinfra"
)

var dbCfg config.DatabaseConfig

func TestMain(m *testing.M) {
	postgres, err := testinfra.RunPostgres()
	if err != nil {
		log.Fatalf("could not start postgres container: %v", err)
	}

	dbCfg = postgres.Config

	m.Run()

	if err = postgres.Shutdown(); err != nil {
		log.Fatalf("could not teardown postgres container: %v", err)
	}
}

func TestNew(t *testing.T) {
	srv, err := New(dbCfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	t.Cleanup(func() { _ = srv.Close() })
}

func TestHealth(t *testing.T) {
	srv, err := New(dbCfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	stats := srv.Health(context.Background())

	if stats["status"] != "up" {
		t.Fatalf("expected status to be up, got %s", stats["status"])
	}

	if _, ok := stats["error"]; ok {
		t.Fatalf("expected error not to be present")
	}

	if stats["message"] != "It's healthy" {
		t.Fatalf("expected message to be 'It's healthy', got %s", stats["message"])
	}
}

func TestClose(t *testing.T) {
	srv, err := New(dbCfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if srv.Close() != nil {
		t.Fatalf("expected Close() to return nil")
	}
}
