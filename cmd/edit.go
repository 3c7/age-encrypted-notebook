package main

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
	"github.com/3c7/aen/internal/utils"
)

// editNode, sililar to createNote, decrypts and writes a note to a temporary file which then can be edited through the configured editor.
func editNote(pathFlag, keyFlag, slugFlag string, idFlag uint, editorCmd []string, shredFlag bool) {
	var note *model.EncryptedNote
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if len(slugFlag) > 0 {
		note, err = db.GetEncryptedNoteBySlug(slugFlag)
		if err != nil {
			log.Fatalf("Error receiving note %s from DB: %v", slugFlag, err)
		}
	} else if idFlag > 0 {
		note, err = db.GetEncryptedNoteByIndex(idFlag)
		if err != nil {
			log.Fatalf("Error receiving note %d from DB: %v", idFlag, err)
		}
	} else {
		log.Fatal("Error receiving note from DB: either slug or id must be given.")
	}

	if note.IsFile {
		log.Fatalf("Editing binary notes is not implemented.")
	}

	identity, err := utils.IdentityFromKeyfile(keyFlag)
	if err != nil {
		log.Fatalf("Could not load private key: %v", err)
	}

	decryptedNote, err := note.ToDecryptedNote(identity)
	if err != nil {
		log.Fatalf("Could not decrypt note %s: %v", note.Slug(), err)
	}

	file, err := os.CreateTemp("", "note")
	if err != nil {
		log.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	err = decryptedNote.ToFile(file.Name())
	if err != nil {
		log.Fatalf("Error writing note to file %s: %v", file.Name(), err)
	}

	editorCmd = append(editorCmd, file.Name())
	Cmd := exec.Command(editorCmd[0], editorCmd[1:]...)
	err = Cmd.Run()
	if err != nil {
		log.Fatalf("Error running command: %v", err)
	}

	newNote, err := model.NotefileToNote(file.Name())
	if err != nil {
		log.Fatalf("Error reading note from file %s: %v", file.Name(), err)
	}

	newNote.Uuid = note.Uuid
	newNote.Time = time.Now()
	if newNote.Slug() != note.Slug() {
		err = db.DeleteNoteBySlug(note.Slug())
		if err != nil {
			log.Fatalf("Could not delete old note by slug %s: %v", note.Slug(), err)
		}
	}
	recipients, err := db.GetAgeRecipients()
	if err != nil {
		log.Fatalf("Could not get recipients: %v", err)
	}
	newEncryptedNote, err := newNote.ToEncryptedNote(recipients...)
	if err != nil {
		log.Fatalf("Could not encrypt note: %v", err)
	}
	err = db.SaveEncryptedNote(&newEncryptedNote)
	if err != nil {
		log.Fatalf("Could not save encrypted note: %v", err)
	}
	log.Printf("Written note %s.", newEncryptedNote.Slug())
	if shredFlag {
		utils.OverwriteFileContent(file.Name())
	}
}
