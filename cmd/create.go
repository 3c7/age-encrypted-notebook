package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/utils"
)

// createNote creates a new note through
// - creating a temporary file
// - opening the file with the configured editor
// - wait until the process exits
// - read the file
// - use the first line as title and the remaining content as note text
func createNote(pathFlag string, cmdString []string, shredFlag bool) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database file: %v", err)
	}
	db.Close()

	tmpfile, err := ioutil.TempFile("", "note")
	if err != nil {
		log.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	cmdString = append(cmdString, tmpfile.Name())

	Cmd := exec.Command(cmdString[0], cmdString[1:]...)
	err = Cmd.Run()
	if err != nil {
		log.Fatalf("Error running command: %v", err)
	}
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		log.Fatalf("Error reading temporary file %s: %v", tmpfile.Chdir(), err)
	}
	slicedData := strings.Split(string(data), "\n")
	regex, err := regexp.Compile("[^a-zA-Z0-9 !\"§$%&/()=]+")
	if err != nil {
		log.Fatalf("Error compiling regular expression: %v", err)
	}
	title := regex.ReplaceAllString(slicedData[0], "")
	message := strings.Join(slicedData[1:], "\n")
	if len(title) == 0 || len(message) == 0 {
		log.Fatalf("Error creating note: both, title and message, must be given.")
	}
	if shredFlag {
		utils.OverwriteFileContent(tmpfile.Name())
	}
	writeNote(pathFlag, title, message)
}
