package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/3c7/aen"
	"github.com/3c7/aen/internal/model"
	"github.com/3c7/aen/internal/utils"
)

var Version string

const usage string = `Age Encrypted Notebook $(VERSION)

Write age encrypted text snippets ("notes") into a Bolt database.

Subcommands:
  help        (?)   (-b|--brief)
  init        (in)  (-o|--output) <DB path> (-k|--key) <key path>
  list        (ls)  (-d|--db) <DB path>
  create      (cr)  (-d|--db) <DB path> (-S|--shred)
  edit        (ed)  (-d|--db) <DB path> (-k|--key) <key path>
                    (-s|--slug) <slug> (-i|--id) <id> (-S|--shred)
  write       (wr)  (-d|--db) <DB path> (-t|--title) <title> (-m|--message) <message>
  get         (g)   (-d|--db) <DB path> (-k|--key) <key path>
                    (-s|--slug) <slug> (-i|--id) <id> (-r|--raw)
  remove      (rm)  (-d|--db) <DB path> (-s|--slug) <slug> (-i|--id) <id>
  recipients  (re)  (-d|--db) <DB path>

More details via "aen help" or with parameter "--help".
`

const help string = `Age Encrypted Notebook $(VERSION)

* DB and keyfile paths can also be given via evironment variables AENDB and AENKEY.
** The default editor can be changed through setting the environment variable AENEDITOR.

Usage:

aen init (in)          Initializes the private key and the database if not already given
                       and adds the own public key to the database
  -o, --output         - Path to DB *
  -k, --key            - Path to age keyfile *

aen list (ls)          Lists the slugs of available notes sorted by their timestamp
  -d, --db             - Path to DB *

aen create (cr)        Creates a new note with an editor using the first line of the created
                       note as title
                       By default the command calls 'codium -w' **
  -d, --db             - Path to DB *
  -S, --shred          - Overwrites temporary file with random data

aen edit (ed)          Edits a note given by slug or id
                       By default the command calls 'codium -w' **
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get
  -S, --shred          - Overwrites temporary file with random data

aen write (wr)         Writes a new note
  -d, --db             - Path to DB *
  -t, --title          - Title of the note
  -m, --message        - Message of the note

aen get (g)            Get and decrypt a note by its slug or id
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get
  -r, --raw            - Only print note content without any metadata

aen remove (rm)        Removes note by its slug or id from the database
                       NOTE: While the note is not retrievable through aen anymore,
                       the data reside in the database file until its overwritten by a new note.
  -d, --db             - Path to DB *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get

aen recipients (re)   Lists all recipients and their aliases
  -d, --db            - Path to DB *
`

func main() {
	if buildinfo, ok := debug.ReadBuildInfo(); ok && Version == "" {
		Version = buildinfo.Main.Version
	} else if Version == "" {
		Version = "(unknown)"
	}

	log.SetFlags(0)
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s\n", strings.Replace(usage, "$(VERSION)", Version, 1)) }
	DetailedUsage := func() { fmt.Fprintf(os.Stderr, "%s\n", strings.Replace(help, "$(VERSION)", Version, 1)) }

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	argString := strings.Join(os.Args, "")
	if strings.Contains(argString, "--help") {
		DetailedUsage()
		os.Exit(1)
	}

	var (
		pathFlag, keyFlag, titleFlag, messageFlag, slugFlag, aliasFlag string
		pathEnv, keyEnv, editorEnv                                     string
		editorCmd                                                      []string
		idFlag                                                         uint
		briefFlag, shredFlag, rawFlag                                  bool
	)

	HelpCmd := flag.NewFlagSet("help", flag.ExitOnError)
	HelpCmd.BoolVar(&briefFlag, "brief", false, "Shows only brief usage information.")
	HelpCmd.BoolVar(&briefFlag, "b", false, "Shows only brief usage information.")

	InitCmd := flag.NewFlagSet("init", flag.ExitOnError)
	InitCmd.StringVar(&pathFlag, "output", "", "Filepath to database file which will be created, if not already available.")
	InitCmd.StringVar(&pathFlag, "o", "", "Filepath to database file which will be created, if not already available.")
	InitCmd.StringVar(&keyFlag, "key", "", "Filepath to key file, will be created if not available.")
	InitCmd.StringVar(&keyFlag, "k", "", "Filepath to key file, will be created if not available.")
	InitCmd.StringVar(&aliasFlag, "alias", "", "Alias to be used for the public key.")
	InitCmd.StringVar(&aliasFlag, "a", "", "Alias to be used for the public key.")

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
	GetCmd.BoolVar(&rawFlag, "raw", false, "Only print note content")
	GetCmd.BoolVar(&rawFlag, "r", false, "Only print note content")

	CreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	CreateCmd.StringVar(&pathFlag, "db", "", "Path to database")
	CreateCmd.StringVar(&pathFlag, "d", "", "Path to database")
	CreateCmd.BoolVar(&shredFlag, "shred", false, "Shred file contents afterwards")
	CreateCmd.BoolVar(&shredFlag, "S", false, "Shred file contents afterwards")

	RmCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	RmCmd.StringVar(&pathFlag, "db", "", "Path to database")
	RmCmd.StringVar(&pathFlag, "d", "", "Path to database")
	RmCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	RmCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	RmCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	RmCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	RmCmd.UintVar(&idFlag, "id", 0, "ID for note")
	RmCmd.UintVar(&idFlag, "i", 0, "ID for note")

	EditCmd := flag.NewFlagSet("edit", flag.ExitOnError)
	EditCmd.StringVar(&pathFlag, "db", "", "Path to database")
	EditCmd.StringVar(&pathFlag, "d", "", "Path to database")
	EditCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	EditCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	EditCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	EditCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	EditCmd.UintVar(&idFlag, "id", 0, "ID for note")
	EditCmd.UintVar(&idFlag, "i", 0, "ID for note")
	EditCmd.BoolVar(&shredFlag, "shred", false, "Shred file contents afterwards")
	EditCmd.BoolVar(&shredFlag, "S", false, "Shred file contents afterwards")

	RecipientsCmd := flag.NewFlagSet("recipients", flag.ExitOnError)
	RecipientsCmd.StringVar(&pathFlag, "db", "", "Path to database")
	RecipientsCmd.StringVar(&pathFlag, "d", "", "Path to database")

	pathEnv = os.Getenv("AENDB")
	keyEnv = os.Getenv("AENKEY")
	editorEnv = os.Getenv("AENEDITOR")

	if len(editorEnv) > 0 {
		editorCmd = strings.Split(editorEnv, " ")
	} else {
		editorCmd = strings.Split("codium -w", " ")
	}

	switch os.Args[1] {
	case "help", "he", "?":
		HelpCmd.Parse(os.Args[2:])
		if briefFlag {
			flag.Usage()
		} else {
			DetailedUsage()
		}

	case "init", "in":
		InitCmd.Parse(os.Args[2:])
		path, key, err := utils.GetPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}
		err = initAen(path, key, aliasFlag)
		if err != nil {
			log.Fatalf("Error initializing aen: %v", err)
		}

	case "list", "ls":
		ListCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error listing notes: %v", err)
		}
		listNotes(path)

	case "write", "wr":
		WriteCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error writing note: %v", err)
		}

		// Check if this program is used in a unix pipe and read from stdin, if this is the case
		if utils.IsPipe() {
			messageFlag = ""
			log.Println("Using text from stdin as message.")
			reader := bufio.NewReader(os.Stdin)
			var err error = nil
			var s string
			for err == nil {
				s, err = reader.ReadString('\n')
				messageFlag = messageFlag + s + "\n"
			}
		}

		if len(titleFlag) == 0 {
			log.Fatal("Error writing note: title must be given.")
		}

		if len(messageFlag) == 0 {
			log.Fatal("Error writing note: message must be given.")
		}
		writeNote(path, titleFlag, messageFlag)

	case "get", "g":
		GetCmd.Parse(os.Args[2:])
		path, key, err := utils.GetPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error getting note: %v", err)
		}
		if len(slugFlag) == 0 && idFlag == 0 {
			log.Fatal("Error getting note: ID or Slug must be given.")
		}
		getNote(path, key, slugFlag, idFlag, rawFlag)

	case "create", "cr":
		CreateCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error creating note: %v", err)
		}
		createNote(path, editorCmd, shredFlag)

	case "edit", "ed":
		EditCmd.Parse(os.Args[2:])
		path, key, err := utils.GetPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error editing note: %v", err)
		}
		editNote(path, key, slugFlag, int(idFlag), editorCmd, shredFlag)

	case "remove", "del", "rm":
		RmCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error deleting note: %v", err)
		}
		deleteNote(path, slugFlag, idFlag)

	case "version", "ver", "v":
		log.Printf("Age Encrypted Notebook version: %s", Version)

	case "recipients", "re":
		RecipientsCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error listing recipients: %v", err)
		}
		listRecipients(path)

	default:
		flag.Usage()
		log.Fatalf("Subcommand unknown: %s", os.Args[1])
	}
}

// Initializes AEN with a database and a key.
// If database is already available, a key will be generated.
// If both are available, the public key will be added as recipient.
func initAen(path string, keyPath string, aliasFlag string) (err error) {
	key, err := aen.EnsureKey(keyPath)
	if err != nil {
		return err
	}
	fmt.Printf("Public key: %s\n", key.Recipient().String())

	db, err := aen.OpenDatabase(path, true)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(aliasFlag) == 0 {
		aliasFlag = fmt.Sprintf("%x", crc32.ChecksumIEEE([]byte(key.Recipient().String())))
	}

	recipient := model.Recipient{
		Alias:     aliasFlag,
		Publickey: key.Recipient().String(),
	}

	err = db.AddRecipient(recipient)
	return
}

// List all notes available in the database and print them ordered by the creation time.
func listNotes(pathFlag string) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database file: %v", err)
	}
	defer db.Close()

	notes, err := db.GetEncryptedNotes()
	if err != nil {
		log.Fatalf("Error reading notes: %v", err)
	}
	if len(notes) == 0 {
		log.Println("No notes available.")
		return
	}
	model.SortNoteSlice(notes)
	fmt.Printf("| %-5s | %-25s | %-25s | %-25s |\n", "ID", "Title", "Creation time", "Slug")
	var title string
	for idx, note := range notes {
		if len(note.Title) > 25 {
			title = note.Title[:22] + "..."
		} else {
			title = note.Title
		}
		fmt.Printf("| %-5s | %-25s | %-25s | %-25s |\n", fmt.Sprintf("%d", idx+1), title, note.Time.Format("2006-01-02 15:04:05"), note.Slug())
	}
}

// Creates a new note through
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

// Sililar to createNote this function decrypts and writes a note to a temporary file
// which then can be edited through the configured editor.
func editNote(pathFlag, keyFlag, slugFlag string, idFlag int, editorCmd []string, shredFlag bool) {
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

// Writes a new note based on the parameters given.
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

func getNote(pathFlag, keyFlag, slugFlag string, idFlag uint, rawFlag bool) {
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
		encryptedNote, err = db.GetEncryptedNoteByIndex(int(idFlag))
		if err != nil {
			log.Fatalf("Could not load note by id: %v", err)
		}
	}

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
	}
}

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
		note, err = db.GetEncryptedNoteByIndex(int(idFlag))
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

func listRecipients(pathFlag string) {
	db, err := aen.OpenDatabase(pathFlag, false)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	recipients, err := db.GetRecipients()
	if err != nil {
		log.Fatalf("Error loading recipients: %v", err)
	}
	if len(recipients) == 0 {
		// Should not really be the case, but anyway...
		log.Println("Recipient list is empty.")
	} else {
		log.Printf("| %-20s | %-62s |", "Alias", "Public Key")
		for _, r := range recipients {
			log.Printf("| %-20s | %-62s |", r.Alias, r.Publickey)
		}
	}
}
