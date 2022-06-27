package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	gopath "path"
	"regexp"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	uuid "github.com/google/uuid"
)

type Recipient struct {
	Alias     string
	Publickey string
}

func NewRecipient(alias string, pubkey string) (recipient *Recipient) {
	return &Recipient{
		Alias:     alias,
		Publickey: pubkey,
	}
}

func NewRecipientFromIdentity(alias string, identity age.X25519Identity) (recipient *Recipient) {
	return &Recipient{
		Alias:     alias,
		Publickey: identity.Recipient().String(),
	}
}

func (r *Recipient) Json() (j []byte, err error) {
	return json.Marshal(r)
}

type Note struct {
	Uuid  uuid.UUID
	Time  time.Time
	Title string
	Text  string
}

type FileNote struct {
	Note
	Content []byte
}

func NewNote(title string, text string) (note *Note) {
	return &Note{
		uuid.New(),
		time.Now(),
		title,
		text,
	}
}

func NewFileNote(title string, content []byte) (note *FileNote) {
	textNote := NewNote(title, "")
	return &FileNote{
		*textNote,
		content,
	}
}

func (note *Note) Slug() (slug string) {
	regex, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Fatalf("Error compiling regular expression: %v", err)
	}

	slug = regex.ReplaceAllString(note.Title, "")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ToLower(slug)
	return slug
}

// Reads a file and parses it's content. The resulting Note does not have a UUID or a Time set.
func NotefileToNote(path string) (note *Note, err error) {
	if _, err = os.Stat(path); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	return &Note{
		Title: lines[0],
		Text:  strings.Join(lines[1:], "\n"),
	}, nil
}

func FileNoteFromFile(path string, title string) (bNote *FileNote, err error) {
	if _, err = os.Stat(path); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if title == "" {
		title = gopath.Base(path)
	}
	return &FileNote{
		Note: Note{
			Uuid:  uuid.New(),
			Title: title,
			Time:  time.Now(),
		},
		Content: content,
	}, nil
}

func (note *Note) Encrypt(x25519recipients ...age.X25519Recipient) (ciphertext string, err error) {
	var recipients []age.Recipient
	for r := range x25519recipients {
		recipients = append(recipients, &x25519recipients[r])
	}
	out := &bytes.Buffer{}
	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		log.Fatalf("Failed to create encrypted note with uuid %s.", note.Uuid.String())
	}
	if _, err := io.WriteString(w, note.Text); err != nil {
		log.Fatalf("Failed to encrypt data for note %s: %v", note.Uuid.String(), err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("Error on closing encrypted note with uuid %s.", note.Uuid.String())
	}
	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

func (note *Note) ToEncryptedNote(x25519recipients ...age.X25519Recipient) (encryptedNote EncryptedNote, err error) {
	ciphertext, err := note.Encrypt(x25519recipients...)
	return EncryptedNote{
		note.Uuid,
		note.Time,
		note.Title,
		ciphertext,
		false,
	}, err
}

func (note *Note) Json() (encodedNote []byte, err error) {
	return json.Marshal(note)
}

// Writes note to a file in the format of Title\nText.
func (note *Note) ToFile(path string) (err error) {
	content := strings.Join([]string{note.Title, note.Text}, "\n")
	return os.WriteFile(path, []byte(content), 0600)
}

func (bNote *FileNote) Encrypt(x25519recipients ...age.X25519Recipient) (ciphertext string, err error) {
	var recipients []age.Recipient
	for r := range x25519recipients {
		recipients = append(recipients, &x25519recipients[r])
	}
	out := &bytes.Buffer{}
	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		log.Fatalf("Failed to create encrypted note with uuid %s.", bNote.Uuid.String())
	}
	if _, err = w.Write(bNote.Content); err != nil {
		log.Fatalf("Failed to encrypt data for note %s: %v", bNote.Uuid.String(), err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("Error on closing encrypted note with uuid %s.", bNote.Uuid.String())
	}
	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

func (bNote *FileNote) ToEncryptedNote(x25519recipients ...age.X25519Recipient) (encryptedNote EncryptedNote, err error) {
	ciphertext, err := bNote.Encrypt(x25519recipients...)
	return EncryptedNote{
		bNote.Uuid,
		bNote.Time,
		bNote.Title,
		ciphertext,
		true,
	}, err
}

type EncryptedNote struct {
	Uuid       uuid.UUID
	Time       time.Time
	Title      string
	Ciphertext string
	IsFile     bool
}

func (encryptedNote *EncryptedNote) Slug() (slug string) {
	regex, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Fatalf("Error compiling regular expression: %v", err)
	}

	slug = regex.ReplaceAllString(encryptedNote.Title, "")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ToLower(slug)
	return slug
}

func (encryptedNote EncryptedNote) Decrypt(identity age.Identity) (text string, err error) {
	var decoded []byte
	if decoded, err = base64.StdEncoding.DecodeString(encryptedNote.Ciphertext); err != nil {
		log.Fatalf("Error decoding encrypted note's ciphertext: %v.", err)
	}
	r, err := age.Decrypt(bytes.NewReader(decoded), identity)
	if err != nil {
		return "", err
	}
	buffer := &bytes.Buffer{}
	if _, err := io.Copy(buffer, r); err != nil {
		log.Fatalf("Could not decrypt note with uuid %s.", encryptedNote.Uuid.String())
	}
	return buffer.String(), nil
}

func (encryptedNote EncryptedNote) DecryptContent(identity age.Identity) (content []byte, err error) {
	var decoded []byte
	if decoded, err = base64.StdEncoding.DecodeString(encryptedNote.Ciphertext); err != nil {
		log.Fatalf("Error decoding encrypted note's ciphertext: %v.", err)
	}
	r, err := age.Decrypt(bytes.NewReader(decoded), identity)
	if err != nil {
		return []byte(""), err
	}
	if content, err = io.ReadAll(r); err != nil {
		log.Fatalf("Could not decrypt note with uuid %s.", encryptedNote.Uuid.String())
	}
	return
}

func (encryptedNote EncryptedNote) ToDecryptedNote(identity age.Identity) (note Note, err error) {
	if encryptedNote.IsFile {
		return Note{}, errors.New("the given note contains a file, therefore ToDecryptedFileNote must be used")
	}

	text, err := encryptedNote.Decrypt(identity)
	return Note{
		encryptedNote.Uuid,
		encryptedNote.Time,
		encryptedNote.Title,
		text,
	}, err
}

func (encryptedNote EncryptedNote) ToDecryptedFileNote(identity age.Identity) (bNote FileNote, err error) {
	if !encryptedNote.IsFile {
		return FileNote{}, errors.New("the given note does not contain a file, please use ToDecryptedNote for decrypting text only notes")
	}
	content, err := encryptedNote.DecryptContent(identity)
	return FileNote{
		Note{
			encryptedNote.Uuid,
			encryptedNote.Time,
			encryptedNote.Title,
			"",
		},
		content,
	}, err
}

func (encryptedNote EncryptedNote) Json() (encodedNote []byte, err error) {
	return json.Marshal(encryptedNote)
}

func SortNoteSlice(notes []EncryptedNote) []EncryptedNote {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Time.After(notes[j].Time)
	})
	return notes
}
