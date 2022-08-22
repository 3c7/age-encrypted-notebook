package main

import (
	"log"
	"strings"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// manipulateTags adds or remove Tags from notes.
func manipulateTags(pathFlag string, idFlag uint, slugFlag, tagAddFlag, tagRemoveFlag string) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	var note *model.EncryptedNote
	if idFlag > 0 {
		note, err = db.GetEncryptedNoteByIndex(idFlag)
		if err != nil {
			log.Fatalf("Note with id %d not available", idFlag)
		}
	} else if len(slugFlag) > 0 {
		note, err = db.GetEncryptedNoteBySlug(slugFlag)
		if err != nil {
			log.Fatalf("Note with slug %s not available", slugFlag)
		}
	} else {
		log.Fatalf("Either ID or Slug must be given.")
	}

	if len(tagAddFlag) > 0 {
		tags := strings.Split(tagAddFlag, ",")
		for i := range tags {
			note.AddTag(strings.TrimSpace(tags[i]))
		}
	}
	if len(tagRemoveFlag) > 0 {
		tags := strings.Split(tagRemoveFlag, ",")
		for i := range tags {
			tag := strings.TrimSpace(tags[i])
			err = note.RemoveTag(tag)
			if err != nil {
				log.Fatalf("Error while removing tag \"%s\": %v", tag, err)
			}
		}
	}
	err = db.SaveEncryptedNote(note)
	if err != nil {
		log.Fatalf("Error saving note: %v", err)
	}
}
