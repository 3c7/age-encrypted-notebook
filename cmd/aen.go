package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strings"

	"filippo.io/age"
	"github.com/3c7/aen/internal/database"
	"github.com/3c7/aen/internal/model"
)

var Version string

const usage string = `Age Encrypted Notebook $(VERSION)

* DB and keyfile paths can also be given via evironment variables AENDB and AENKEY.
** The default editor can be changed setting the environment variable AENEDITOR.

Usage:

aen init          Initializes the private key and the database if not already given and adds the own public key to the database
  -o, --output    - Path to DB *
  -k, --key       - Path to age keyfile *

aen list          Lists the slugs of available notes sorted by their timestamp
  -d, --db        - Path to DB *

aen create        Creates a new note with an editor - by default the command calls 'codium -w' **
  -d, --db        - Path to DB *

aen write         Writes a new note
  -d, --db        - Path to DB *
  -t, --title     - Title of the note
  -m, --message   - Message of the note

aen get           Get and decrypt a note by its slug or id
  -d, --db        - Path to DB *
  -k, --key       - Path to age keyfile *
  -s, --slug      - Slug of note to get
  -i, --id        - ID of note to get

aen del           Delete note by its slug or id
  -d, --db        - Path to DB *
  -s, --slug      - Slug of note to get
  -i, --id        - ID of note to get
`

func main() {
	if buildinfo, ok := debug.ReadBuildInfo(); ok && Version == "" {
		Version = buildinfo.Main.Version
	} else if Version == "" {
		Version = "(unknown)"
	}

	log.SetFlags(0)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s\n", strings.Replace(usage, "$(VERSION)", Version, 1)) }

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	argString := strings.Join(os.Args, "")
	if strings.Contains(argString, "--help") {
		flag.Usage()
		os.Exit(1)
	}

	var (
		pathFlag, keyFlag, titleFlag, messageFlag, slugFlag string
		pathEnv, keyEnv, editorEnv                          string
		editorCmd                                           []string
		idFlag                                              uint
	)

	InitCmd := flag.NewFlagSet("init", flag.ExitOnError)
	InitCmd.StringVar(&pathFlag, "output", "", "Filepath to database file which will be created, if not already available.")
	InitCmd.StringVar(&pathFlag, "o", "", "Filepath to database file which will be created, if not already available.")
	InitCmd.StringVar(&keyFlag, "key", "", "Filepath to key file, will be created if not available.")
	InitCmd.StringVar(&keyFlag, "k", "", "Filepath to key file, will be created if not available.")

	ListCmd := flag.NewFlagSet("list", flag.ExitOnError)
	ListCmd.StringVar(&pathFlag, "db", "", "Path to database")
	ListCmd.StringVar(&pathFlag, "d", "", "Path to database")

	WriteCmd := flag.NewFlagSet("write", flag.ExitOnError)
	WriteCmd.StringVar(&pathFlag, "db", "", "Path to database")
	WriteCmd.StringVar(&pathFlag, "d", "", "Path to database")
	WriteCmd.StringVar(&titleFlag, "title", "", "Title for test writing a note.")
	WriteCmd.StringVar(&titleFlag, "t", "", "Title for test writing a note.")
	WriteCmd.StringVar(&messageFlag, "message", "", "Content for writing a note.")
	WriteCmd.StringVar(&messageFlag, "m", "", "Content for writing a note.")

	GetCmd := flag.NewFlagSet("get", flag.ExitOnError)
	GetCmd.StringVar(&pathFlag, "db", "", "Path to database")
	GetCmd.StringVar(&pathFlag, "d", "", "Path to database")
	GetCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	GetCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	GetCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	GetCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	GetCmd.UintVar(&idFlag, "id", 0, "ID for note")
	GetCmd.UintVar(&idFlag, "i", 0, "ID for note")

	CreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	CreateCmd.StringVar(&pathFlag, "db", "", "Path to database")
	CreateCmd.StringVar(&pathFlag, "d", "", "Path to database")

	DelCmd := flag.NewFlagSet("del", flag.ExitOnError)
	DelCmd.StringVar(&pathFlag, "db", "", "Path to database")
	DelCmd.StringVar(&pathFlag, "d", "", "Path to database")
	DelCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	DelCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	DelCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	DelCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	DelCmd.UintVar(&idFlag, "id", 0, "ID for note")
	DelCmd.UintVar(&idFlag, "i", 0, "ID for note")

	pathEnv = os.Getenv("AENDB")
	keyEnv = os.Getenv("AENKEY")
	editorEnv = os.Getenv("AENEDITOR")

	if len(editorEnv) > 0 {
		editorCmd = strings.Split(editorEnv, " ")
	} else {
		editorCmd = strings.Split("codium -w", " ")
	}

	switch os.Args[1] {
	case "init":
		InitCmd.Parse(os.Args[2:])
		path, key, err := getPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}
		err = initAen(path, key)
		if err != nil {
			log.Fatalf("Error initializing aen: %v", err)
		}

	case "list":
		ListCmd.Parse(os.Args[2:])
		path, _, err := getPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error listing notes: %v", err)
		}
		listNotes(path)

	case "write":
		WriteCmd.Parse(os.Args[2:])
		path, _, err := getPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error writing note: %v", err)
		}
		if len(titleFlag) == 0 || len(messageFlag) == 0 {
			log.Fatal("Error writing note: title and message must be given.")
		}
		writeNote(path, titleFlag, messageFlag)

	case "get":
		GetCmd.Parse(os.Args[2:])
		path, key, err := getPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error getting note: %v", err)
		}
		if len(slugFlag) == 0 && idFlag == 0 {
			log.Fatal("Error getting note: ID or Slug must be given.")
		}
		getNote(path, key, slugFlag, idFlag)

	case "create":
		CreateCmd.Parse(os.Args[2:])
		path, _, err := getPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error creating note: %v", err)
		}
		createNote(path, editorCmd)

	case "del":
		DelCmd.Parse(os.Args[2:])
		path, _, err := getPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error deleting note: %v", err)
		}
		deleteNote(path, slugFlag, idFlag)

	case "version", "ver":
		log.Printf("Age Encrypted Notebook version: %s", Version)

	default:
		flag.Usage()
		log.Fatalf("Subcommand unknown: %s", os.Args[1])
	}
}

// Initializes AEN with a database and a key. If database is already available, a key will be generated.
// If both are available, the public key will be added as recipient.
func initAen(path string, keyPath string) (err error) {
	key, err := ensureKey(keyPath)
	if err != nil {
		return err
	}
	log.Printf("Public key: %s\n", key.Recipient().String())

	db, err := ensureDatabase(path)
	if err != nil {
		return err
	}

	err = db.Open()
	if err != nil {
		return err
	}

	err = db.AddRecipient(key.Recipient())
	return
}

// Checks for paths given as parameters and envs. Returns the correct path prioritizing parameters over envs.
func getPaths(pathFlag, pathEnv, keyFlag, keyEnv string, keyNeeded bool) (path, key string, err error) {
	if len(pathFlag) > 0 {
		path = pathFlag
	} else if len(pathEnv) > 0 {
		path = pathEnv
	} else {
		return "", "", errors.New("path to database must be given")
	}

	if len(keyFlag) > 0 {
		key = keyFlag
	} else if len(keyEnv) > 0 {
		key = keyEnv
	} else {
		if keyNeeded {
			return "", "", errors.New("path to keyfile must be given")
		}
	}
	return path, key, nil
}

// Checks if a keyfile is available and tries to parse it. If no file is available, generate a new key.
func ensureKey(keyPath string) (key *age.X25519Identity, err error) {
	if _, err := os.Stat(keyPath); err == nil {
		buf, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		content := string(buf)
		if strings.Contains(content, "\n") {
			content = strings.ReplaceAll(content, "\n", "")
		}
		key, err = age.ParseX25519Identity(content)
		return key, err
	} else if errors.Is(err, os.ErrNotExist) {
		key, err = age.GenerateX25519Identity()
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(keyPath, []byte(fmt.Sprintf("%s\n", key.String())), 0600|os.ModeExclusive)
		if err != nil {
			return nil, err
		}
		log.Printf("Written key to %s.", keyPath)
		return key, err
	} else {
		return nil, err
	}
}

// Creates new database if not available, but returns error if access is denied.
func ensureDatabase(path string) (db *database.Database, err error) {
	if _, err := os.Stat(path); err == nil || errors.Is(err, os.ErrNotExist) {
		return database.NewDatabaseInstance(path), nil
	} else {
		return nil, err
	}
}

// List all notes available in the database.
func listNotes(pathFlag string) {
	if _, err := os.Stat(pathFlag); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Database %s not available.\n", pathFlag)
	}
	db := database.NewDatabaseInstance(pathFlag)
	err := db.Open()
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	notes, err := db.GetEncryptedNotes()
	if err != nil {
		log.Fatalf("Error reading notes: %v", err)
	}
	if len(notes) == 0 {
		log.Println("No notes available.")
		return
	}
	model.SortNoteSlice(notes)
	log.Printf("| %-5s | %-25s | %-25s | %-25s |\n", "ID", "Title", "Creation time", "Slug")
	var title string
	for idx, note := range notes {
		if len(note.Title) > 25 {
			title = note.Title[:22] + "..."
		} else {
			title = note.Title
		}
		log.Printf("| %-5s | %-25s | %-25s | %-25s |\n", fmt.Sprintf("%d", idx+1), title, note.Time.Format("2006-01-02 15:04:05"), note.Slug())
	}
}

// Creates a new note through calling 'codium -w'
// Todo: Make editor customizable
func createNote(pathFlag string, cmdString []string) {
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
	writeNote(pathFlag, title, message)
}

// Writes a new note
func writeNote(pathFlag, titleFlag, messageFlag string) {
	db := database.NewDatabaseInstance(pathFlag)
	err := db.Open()
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	note := model.NewNote(titleFlag, messageFlag)
	x25519Recipients, err := db.GetReceipients()
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

func getNote(pathFlag, keyFlag, slugFlag string, idFlag uint) {
	var encryptedNote *model.EncryptedNote
	db := database.NewDatabaseInstance(pathFlag)
	err := db.Open()
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}

	keyFile, err := os.ReadFile(keyFlag)
	if err != nil {
		log.Fatalf("Could not read key file: %v", err)
	}
	re, err := regexp.Compile("[^A-Z0-9-]+")
	if err != nil {
		log.Fatalf("Could not parse regex for stripping keyfile: %v", err)
	}
	key := re.ReplaceAllString(string(keyFile), "")

	if slugFlag != "" {
		encryptedNote, err = db.GetEncryptedNoteBySlug(slugFlag)
		if err != nil {
			log.Fatalf("Could not load note by slug: %v", err)
		}
	} else if idFlag != 0 {
		encryptedNote, err = db.GetEncryptedNoteByIndex(int(idFlag))
		if err != nil {
			log.Fatalf("Could not load note by id: %v", err)
		}
	}

	identity, err := age.ParseX25519Identity(key)
	if err != nil {
		log.Fatalf("Could not parse key file content: %v", err)
	}

	note, err := encryptedNote.ToDecryptedNote(identity)
	if err != nil {
		log.Fatalf("Could not decrypt note: %v", err)
	}

	log.Printf("Title: %s (%s)\n", note.Title, note.Uuid.String())
	log.Printf("Created: %s\n", note.Time.Format("2006-01-02 15:04:05"))
	log.Printf("Content:\n%s\n", note.Text)
}

func deleteNote(pathFlag, slugFlag string, idFlag uint) {
	var err error
	var note *model.EncryptedNote
	if len(slugFlag) == 0 && idFlag == 0 {
		log.Fatal("Error deleting note: either slug or id must be given.")
	}

	db := database.NewDatabaseInstance(pathFlag)
	err = db.Open()
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}

	if len(slugFlag) > 0 {
		err = db.DeleteNoteBySlug(slugFlag)
	} else if idFlag > 0 {
		note, err = db.GetEncryptedNoteByIndex(int(idFlag))
		if err != nil {
			log.Fatalf("Couldn't get note by index: %v", err)
		}
		err = db.DeleteNoteBySlug(note.Slug())
	} else {
		err = errors.New("either of slug or id must be given")
	}
	if err != nil {
		log.Fatalf("Could not delete note: %v", err)
	}
}
