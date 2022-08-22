package main

import (
	"fmt"
	"log"
	"os"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
	"github.com/3c7/aen/internal/utils"
)

// getNote receives a note from the database and write it to a file in case its a FileNote
func getNote(pathFlag, keyFlag, slugFlag, fileFlag string, idFlag uint, rawFlag bool) {
	var encryptedNote *model.EncryptedNote
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	identity, err := utils.IdentityFromKeyfile(keyFlag)
	if err != nil {
		log.Fatalf("Could not load private key: %v", err)
	}

	if slugFlag != "" {
		encryptedNote, err = db.GetEncryptedNoteBySlug(slugFlag)
		if err != nil {
			log.Fatalf("Could not load note by slug: %v", err)
		}
	} else if idFlag != 0 {
		encryptedNote, err = db.GetEncryptedNoteByIndex(idFlag)
		if err != nil {
			log.Fatalf("Could not load note by id: %v", err)
		}
	}

	if encryptedNote.IsFile {
		var filename string
		fNote, err := encryptedNote.ToDecryptedFileNote(identity)
		if err != nil {
			log.Fatalf("Could not decrypt note: %v", err)
		}

		if fileFlag == "" {
			filename = fileFlag
		} else {
			filename = fNote.Title
		}

		if err = os.WriteFile(filename, fNote.Content, 0600); err != nil {
			log.Fatalf("Error writing file: %v", err)
		}
		log.Printf("Written file to %s.", fileFlag)
	} else {
		note, err := encryptedNote.ToDecryptedNote(identity)
		if err != nil {
			log.Fatalf("Could not decrypt note: %v", err)
		}

		if rawFlag {
			fmt.Printf("%s\n", note.Text)
		} else {
			fmt.Printf("Title: %s (%s)\n", note.Title, note.Uuid.String())
			fmt.Printf("Created: %s\n", note.Time.Format("2006-01-02 15:04:05"))
			fmt.Printf("Content:\n%s\n", note.Text)
			if len(encryptedNote.Attachments) > 0 {
				fmt.Println("Attachments:")
				for i := range encryptedNote.Attachments {
					fmt.Printf("  - %s\n", encryptedNote.Attachments[i].Filename)
					fmt.Printf("    MD5:\t%s\n", encryptedNote.Attachments[i].Md5)
					fmt.Printf("    SHA1:\t%s\n", encryptedNote.Attachments[i].Sha1)
					fmt.Printf("    SHA256:\t%s\n", encryptedNote.Attachments[i].Sha256)
				}
			}
		}
	}
}
