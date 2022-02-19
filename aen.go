package aen

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/3c7/aen/internal/database"
)

// OpenDatabase returns an instanciated Database struct.
// If the database file is not available, a custom error will be returned.
// However, if the parameter ensure is given, the database will be created.
// Calling this function should be followed with a "defer db.Close()"
func OpenDatabase(path string, ensure bool) (db *database.Database, err error) {
	_, err = os.Stat(path)
	if err == nil {
		db = database.NewDatabaseInstance(path)
	} else if errors.Is(err, os.ErrNotExist) && ensure {
		db = database.NewDatabaseInstance(path)
	} else {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("database file %s not available", path)
		}
		return nil, err
	}
	err = db.Open()
	return db, err
}

// EnsureKey returns a pointer to an age.X25519Identity struct.
// The struct is created through the according parsing function of age.
// If the keyfile is not available, a new keyfile will be generated.
func EnsureKey(path string) (identity *age.X25519Identity, err error) {
	if _, err := os.Stat(path); err == nil {
		buf, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		content := string(buf)
		if strings.Contains(content, "\n") {
			content = strings.ReplaceAll(content, "\n", "")
		}
		identity, err = age.ParseX25519Identity(content)
		return identity, err
	} else if errors.Is(err, os.ErrNotExist) {
		identity, err = age.GenerateX25519Identity()
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(path, []byte(fmt.Sprintf("%s\n", identity.String())), 0600|os.ModeExclusive)
		if err != nil {
			return nil, err
		}
		log.Printf("Written key file to %s.", path)
		return identity, err
	} else {
		return nil, err
	}
}
