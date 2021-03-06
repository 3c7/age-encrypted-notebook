package aen_test

import (
	"os"
	"testing"

	"github.com/3c7/aen"
	"go.etcd.io/bbolt"
)

// Provides a temporary file path
func ProvideTemporaryFilepath(deleted bool) (path string, err error) {
	tmpfile, err := os.CreateTemp("", "database")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	if deleted {
		os.Remove(tmpfile.Name())
	}
	return tmpfile.Name(), nil
}

func TestOpenDatabaseFailure(t *testing.T) {
	tmpfile, err := ProvideTemporaryFilepath(true)
	if err != nil {
		t.Errorf("Error creating temporary file path: %v", err)
	}

	_, err = aen.OpenDatabase(tmpfile, false)
	if err == nil {
		t.Errorf("OpenDatabase should return an error as the file %s was deleted before.", tmpfile)
	}
	t.Logf("OpenDatabase returned expected error: %v", err)
}

func TestOpenDatabaseEnsured(t *testing.T) {
	tmpfile, err := ProvideTemporaryFilepath(true)
	if err != nil {
		t.Errorf("Error creating temporary file path: %v", err)
	}

	db, err := aen.OpenDatabase(tmpfile, true)
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	db.Close()
	os.Remove(tmpfile)
}

func TestOpenDatabase(t *testing.T) {
	tmpfile, err := ProvideTemporaryFilepath(true)
	if err != nil {
		t.Errorf("Could not create temporary file: %v", err)
	}
	db, err := bbolt.Open(tmpfile, 0600, nil)
	if err != nil {
		t.Errorf("Could not create database instance: %v", err)
	}
	db.Close()

	db2, err := aen.OpenDatabase(tmpfile, false)
	if err != nil {
		t.Fatalf("Could not open database file created by bolt.")
	}
	db2.Close()
	os.Remove(tmpfile)
}
