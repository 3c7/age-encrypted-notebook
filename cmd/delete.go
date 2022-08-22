package main

import (
	"errors"
	"log"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// deleteNote deletes a note from the database by slug or id
func deleteNote(pathFlag, slugFlag string, idFlag uint) {
	var err error
	var note *model.EncryptedNote
	if len(slugFlag) == 0 && idFlag == 0 {
		log.Fatal("Error deleting note: either slug or id must be given.")
	}

	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if len(slugFlag) > 0 {
		err = db.DeleteNoteBySlug(slugFlag)
	} else if idFlag > 0 {
		note, err = db.GetEncryptedNoteByIndex(idFlag)
		if err != nil {
			log.Fatalf("Couldn't get note by index: %v", err)
		}
		slugFlag = note.Slug()
		err = db.DeleteNoteBySlug(note.Slug())
	} else {
		err = errors.New("either of slug or id must be given")
	}
	if err != nil {
		log.Fatalf("Could not delete note: %v", err)
	}
	log.Printf("Deleted note %s.", slugFlag)
}
