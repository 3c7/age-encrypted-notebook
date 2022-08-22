package main

import (
	"log"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// attachFile attaches a new file to a note through
// - loading a note by its ID
// - decrypting the note
// - reading the file and put it into the Attachment model
// - encrypting the note again
// - storing the note in the database
func attachFile(dbPath, keyPath, filePath, fileName string, noteId uint) {
	if noteId == 0 {
		log.Fatalf("Note index must be given, but was %d", noteId)
	}
	db, err := aen.OpenDatabase(dbPath, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	encryptedNote, err := db.GetEncryptedNoteByIndex(noteId)
	if err != nil {
		log.Fatalf("Could not get note by id: %v", err)
	}

	attachment, err := model.NewAttachmentFromFile(fileName, filePath)
	if err != nil {
		log.Fatalf("Could not read file %s: %v", filePath, err)
	}

	if given, name := encryptedNote.CheckSha256Hash(attachment.Sha256); given {
		log.Fatalf("Attachment already present under the name %s.", name)
	}

	recipients, err := db.GetAgeRecipients()
	if err != nil {
		log.Fatalf("Could not load recipients: %v", err)
	}

	encryptedAttachment, err := attachment.Encrypt(recipients...)
	if err != nil {
		log.Fatalf("Could not encrypt attachment %s: %v", filePath, err)
	}

	encryptedNote.Attachments = append(encryptedNote.Attachments, *encryptedAttachment)
	if err = db.SaveEncryptedNote(encryptedNote); err != nil {
		log.Fatalf("Could not attach encrypted note: %v", err)
	}
}
