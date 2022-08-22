package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
)

// listNotes lists all notes available in the database and print them ordered by the creation time.
// Additional information, such as flags, are displayed.
func listNotes(pathFlag, tagFlag string, showTagsFlag bool) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database file: %v", err)
	}
	defer db.Close()

	var notes []model.EncryptedNote
	if len(tagFlag) == 0 {
		notes, err = db.GetEncryptedNotes()
		if err != nil {
			log.Fatalf("Error reading notes: %v", err)
		}
	} else {
		notes, err = db.GetEncryptedNoteByTag(tagFlag)
		if err != nil {
			log.Fatalf("Error finding notes: %v", err)
		}
	}
	if len(notes) == 0 {
		log.Println("No notes available.")
		return
	}
	model.SortNoteSlice(notes)
	headers := fmt.Sprintf("| %-5s | %-5s | %-50s |", "Flags", "ID", "Title")
	if showTagsFlag {
		headers += fmt.Sprintf(" %-25s |", "Tags")
	}
	headers += fmt.Sprintf(" %-25s |\n", "Creation time")
	fmt.Print(headers)
	var title string
	for idx, note := range notes {
		if len(note.Title) > 50 {
			title = note.Title[:49] + "..."
		} else {
			title = note.Title
		}
		line := fmt.Sprintf("| %-5s | %-5s | %-50s |", note.Flags(), fmt.Sprintf("%d", idx+1), title)
		if showTagsFlag {
			tags := strings.Join(note.Tags, ", ")
			line += fmt.Sprintf(" %-25s |", tags)
		}
		line += fmt.Sprintf(" %-25s |\n", note.Time.Format("2006-01-02 15:04:05"))
		fmt.Print(line)
	}
}
