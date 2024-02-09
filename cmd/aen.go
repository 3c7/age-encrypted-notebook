package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/3c7/aen/internal/utils"
)

var Version string

const usage string = `Age Encrypted Notebook $(VERSION)

Write age encrypted text snippets ("notes") into a Bolt database.

Subcommands:
  help        (?)   (-b|--brief)

  add         (a)   (-d|--db) <DB path> (-t|--title) <title> (-f|--file) <file path>
  attach      (at)  (-d|--db) <DB path> (-f|--file) <file path> (-n|--name) <file name>
  create      (cr)  (-d|--db) <DB path> (-S|--shred)
  edit        (ed)  (-d|--db) <DB path> (-k|--key) <key path>
                    (-s|--slug) <slug> (-i|--id) <id> (-S|--shred) (-c|--create)
  get         (g)   (-d|--db) <DB path> (-k|--key) <key path>
                    (-s|--slug) <slug> (-i|--id) <id> (-r|--raw)
  init        (in)  (-o|--output) <DB path> (-k|--key) <key path>
  list        (ls)  (-d|--db) <DB path> (-t|--tag) <search tag> --show-tags
  quick       (q)   (-d|--db) <DB path> (-k|--key) <key path>
  recipients  (re)  (-d|--db) <DB path> (-r|--remove) <alias>
  remove      (rm)  (-d|--db) <DB path> (-s|--slug) <slug> (-i|--id) <id>
  tag         (t)   (-d|--db) <DB path> (-s|--slug) <slug> (-i|--id) <id>
                    (-a|--add) <tags> (-r|--remove) <tags>
  write       (wr)  (-d|--db) <DB path> (-t|--title) <title> (-m|--message) <message>

More details via "aen help" or with parameter "--help".
`

const help string = `Age Encrypted Notebook $(VERSION)

* DB and keyfile paths can also be given via environment variables AENDB and AENKEY.
** The default editor can be changed through setting the environment variable AENEDITOR.

Usage:

aen add (a)            Adds a file to the database
  -d, --db             - Path to database
  -t, --title          - Title for the note, default is the filename
  -f, --file           - Path to the file which should be added to the DB

aen attach (at)        Attach a file to a note
  -d, --db             - Path to database
  -f, --file           - Path to file
  -n, --name           - Optional new filename
  -i, --id             - ID of the note to attach file to (see "aen list")

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
  -c, --create         - Create note if not available

aen get (g)            Get and decrypt a note by its slug or id
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get
  -r, --raw            - Only print note content without any metadata

aen init (in)          Initializes the private key and the database if not already given
                       and adds the own public key to the database
  -o, --output         - Path to DB *
  -k, --key            - Path to age keyfile *

aen list (ls)          Lists the slugs of available notes sorted by their timestamp
  -d, --db             - Path to DB *
  -t, --tag            - Only display notes with given tag
  --show-tags          - Display tags

                       The following flags are used:

					   F - File
					   T - Tags
					   A - Attachments

aen quick (q)          Opens the quick note (slug "quicknote"), shreds file on disk per default.
  -d, --db             - Path to DB *
  -k, --key            - Path to age keyfile *

aen recipients (re)    Lists all recipients and their aliases
  -d, --db             - Path to DB *
  -r, --remove         - Remove recipient identified by its alias

aen remove (rm)        Removes note by its slug or id from the database
                       NOTE: While the note is not retrievable through aen anymore,
                       the data reside in the database file until its overwritten by a new note.
  -d, --db             - Path to DB *
  -s, --slug           - Slug of note to get
  -i, --id             - ID of note to get

aen tag (t)            Adds and removes Tags
  -d, --db             - Path to DB *
  -i, --id             - ID of note
  -s, --slug           - Slug of note
  -a, --add            - Comma separated list of tags to add
  -r, --remove         - Comma separated list of tags to remove

aen write (wr)         Writes a new note
  -d, --db             - Path to DB *
  -t, --title          - Title of the note
  -m, --message        - Message of the note
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
		pathFlag, keyFlag, titleFlag, messageFlag, slugFlag, aliasFlag, fileFlag string
		tagAddFlag, tagRemoveFlag, tagFlag                                       string
		pathEnv, keyEnv, editorEnv                                               string
		editorCmd                                                                []string
		idFlag                                                                   uint
		briefFlag, shredFlag, rawFlag, showTagsFlag, createFlag, allFlag         bool
	)

	AddCmd := flag.NewFlagSet("add", flag.ExitOnError)
	AddCmd.StringVar(&pathFlag, "db", "", "Path to database")
	AddCmd.StringVar(&pathFlag, "d", "", "Path to database")
	AddCmd.StringVar(&fileFlag, "file", "", "Path to file")
	AddCmd.StringVar(&fileFlag, "f", "", "Path to file")
	AddCmd.StringVar(&titleFlag, "title", "", "Title of the note (default: filename)")
	AddCmd.StringVar(&titleFlag, "t", "", "Title of the note (default: filename)")

	AttachCmd := flag.NewFlagSet("attach", flag.ExitOnError)
	AttachCmd.StringVar(&pathFlag, "db", "", "Path to database")
	AttachCmd.StringVar(&pathFlag, "d", "", "Path to database")
	AttachCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	AttachCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	AttachCmd.StringVar(&fileFlag, "file", "", "Path to file")
	AttachCmd.StringVar(&fileFlag, "f", "", "Path to file")
	AttachCmd.StringVar(&titleFlag, "name", "", "Optional new filename")
	AttachCmd.StringVar(&titleFlag, "n", "", "Optional new filename")
	AttachCmd.UintVar(&idFlag, "id", 0, "ID for note")
	AttachCmd.UintVar(&idFlag, "i", 0, "ID for note")

	CreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	CreateCmd.StringVar(&pathFlag, "db", "", "Path to database")
	CreateCmd.StringVar(&pathFlag, "d", "", "Path to database")
	CreateCmd.BoolVar(&shredFlag, "shred", false, "Shred file contents afterwards")
	CreateCmd.BoolVar(&shredFlag, "S", false, "Shred file contents afterwards")

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
	EditCmd.BoolVar(&createFlag, "create", false, "Create note if not available")
	EditCmd.BoolVar(&createFlag, "c", false, "Create note if not available")

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
	GetCmd.StringVar(&fileFlag, "output", "", "Path to output file")
	GetCmd.StringVar(&fileFlag, "o", "", "Path to output file")

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
	ListCmd.StringVar(&tagFlag, "tag", "", "Tag to filter for")
	ListCmd.StringVar(&tagFlag, "t", "", "Tag to filter for")
	ListCmd.BoolVar(&allFlag, "all", false, "Tag to filter for")
	ListCmd.BoolVar(&allFlag, "a", false, "Tag to filter for")
	ListCmd.BoolVar(&showTagsFlag, "show-tags", false, "Display tags")

	RecipientsCmd := flag.NewFlagSet("recipients", flag.ExitOnError)
	RecipientsCmd.StringVar(&pathFlag, "db", "", "Path to database")
	RecipientsCmd.StringVar(&pathFlag, "d", "", "Path to database")
	RecipientsCmd.StringVar(&aliasFlag, "remove", "", "Remove recipient with this alias")
	RecipientsCmd.StringVar(&aliasFlag, "r", "", "Remove recipient with this alias")

	RmCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	RmCmd.StringVar(&pathFlag, "db", "", "Path to database")
	RmCmd.StringVar(&pathFlag, "d", "", "Path to database")
	RmCmd.StringVar(&keyFlag, "key", "", "Path to keyfile")
	RmCmd.StringVar(&keyFlag, "k", "", "Path to keyfile")
	RmCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	RmCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	RmCmd.UintVar(&idFlag, "id", 0, "ID for note")
	RmCmd.UintVar(&idFlag, "i", 0, "ID for note")

	TagCmd := flag.NewFlagSet("tag", flag.ExitOnError)
	TagCmd.StringVar(&pathFlag, "db", "", "Path to database")
	TagCmd.StringVar(&pathFlag, "d", "", "Path to database")
	TagCmd.StringVar(&slugFlag, "slug", "", "Slug for note")
	TagCmd.StringVar(&slugFlag, "s", "", "Slug for note")
	TagCmd.StringVar(&tagAddFlag, "add", "", "Comma separated list of tags to add")
	TagCmd.StringVar(&tagAddFlag, "a", "", "Comma separated list of tags to add")
	TagCmd.StringVar(&tagRemoveFlag, "remove", "", "Comma separated list of tags to remove")
	TagCmd.StringVar(&tagRemoveFlag, "r", "", "Comma separated list of tags to remove")
	TagCmd.UintVar(&idFlag, "id", 0, "ID for note")
	TagCmd.UintVar(&idFlag, "i", 0, "ID for note")

	WriteCmd := flag.NewFlagSet("write", flag.ExitOnError)
	WriteCmd.StringVar(&pathFlag, "db", "", "Path to database")
	WriteCmd.StringVar(&pathFlag, "d", "", "Path to database")
	WriteCmd.StringVar(&titleFlag, "title", "", "Title for test writing a note.")
	WriteCmd.StringVar(&titleFlag, "t", "", "Title for test writing a note.")
	WriteCmd.StringVar(&messageFlag, "message", "", "Content for writing a note.")
	WriteCmd.StringVar(&messageFlag, "m", "", "Content for writing a note.")

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
		listNotes(path, tagFlag, showTagsFlag, allFlag)

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
				messageFlag = messageFlag + s
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
		getNote(path, key, slugFlag, fileFlag, idFlag, rawFlag)

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
		editNote(path, key, slugFlag, idFlag, editorCmd, shredFlag, createFlag)

	// Opening a quicknote does basically the same as the edit command with the slug set to quicknote.
	// This is only helpful if the params have been set via ENVs, otherwise this doesn't bring more convenience to the user.
	case "quick", "q":
		EditCmd.Parse(os.Args[2:])
		path, key, err := utils.GetPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error editing note: %v", err)
		}
		editNote(path, key, "quicknote", idFlag, editorCmd, true, true)

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
		listRecipients(path, aliasFlag)

	case "add", "a":
		AddCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error adding file to database: %v", err)
		}
		addFile(path, fileFlag, titleFlag)

	case "tag", "t":
		TagCmd.Parse(os.Args[2:])
		path, _, err := utils.GetPaths(pathFlag, pathEnv, "", "", false)
		if err != nil {
			log.Fatalf("Error manipulating tags: %v", err)
		}
		manipulateTags(path, idFlag, slugFlag, tagAddFlag, tagRemoveFlag)

	case "attach", "at":
		AttachCmd.Parse(os.Args[2:])
		path, key, err := utils.GetPaths(pathFlag, pathEnv, keyFlag, keyEnv, true)
		if err != nil {
			log.Fatalf("Error attaching file: %v", err)
		}
		attachFile(path, key, fileFlag, titleFlag, idFlag)

	default:
		flag.Usage()
		log.Fatalf("Subcommand unknown: %s", os.Args[1])
	}
}
