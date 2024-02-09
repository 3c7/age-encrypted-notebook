package main

import (
	"log"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// addFile adds a file as note to the database
func addFile(pathFlag, fileFlag, titleFlag string) {
	if fileFlag == "" {
		log.Fatal("No file given.")
	}

	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	fNote, err := model.FileNoteFromFile(fileFlag, titleFlag)
	if err != nil {
		log.Fatalf("Error creating binary note from file: %v", err)
	}

	x25519Recipients, err := db.GetAgeRecipients()
	if err != nil {
		log.Fatalf("Error loading recipients: %v", err)
	}
	encryptedNote, err := fNote.ToEncryptedNote(x25519Recipients...)
	if err != nil {
		log.Fatalf("Error during note encryption: %v", err)
	}
	if err = db.SaveEncryptedNote(&encryptedNote); err != nil {
		log.Fatalf("Error adding file to database: %v", err)
	}
}
