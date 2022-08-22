package main

import (
	"fmt"
	"hash/crc32"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// initAen initializes AEN with a database and a key.
// If database is already available, a key will be generated.
// If both are available, the public key will be added as recipient.
func initAen(path string, keyPath string, aliasFlag string) (err error) {
	key, err := aen.EnsureKey(keyPath)
	if err != nil {
		return err
	}
	fmt.Printf("Public key: %s\n", key.Recipient().String())

	db, err := aen.OpenDatabase(path, true)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(aliasFlag) == 0 {
		aliasFlag = fmt.Sprintf("%x", crc32.ChecksumIEEE([]byte(key.Recipient().String())))
	}

	recipient := model.Recipient{
		Alias:     aliasFlag,
		Publickey: key.Recipient().String(),
	}

	err = db.AddRecipient(recipient)
	return
}
