package main

import (
	"log"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// writeNote writes a new note based on the parameters given.
func writeNote(pathFlag, titleFlag, messageFlag string) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	note := model.NewNote(titleFlag, messageFlag)
	x25519Recipients, err := db.GetAgeRecipients()
	if err != nil {
		log.Fatalf("Error loading recipients: %v", err)
	}
	encryptedNote, err := note.ToEncryptedNote(x25519Recipients...)
	if err != nil {
		log.Fatalf("Error during note encryption: %v", err)
	}

	err = db.SaveEncryptedNote(&encryptedNote)
	if err != nil {
		log.Fatalf("Error storing note: %v", err)
	}
	log.Printf("Successfully written note %s.", encryptedNote.Slug())
}
