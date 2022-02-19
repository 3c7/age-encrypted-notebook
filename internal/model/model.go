package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
	uuid "github.com/google/uuid"
)

type Config struct {
	Publickey  string
	Privatekey string
}

func NewConfigFromPrivateKey(privatekey string) *Config {
	identity, err := age.ParseX25519Identity(privatekey)
	if err != nil {
		log.Fatalf("Error parsing identity from private string: %v", err)
	}
	recipient := identity.Recipient()
	return &Config{
		Publickey:  recipient.String(),
		Privatekey: privatekey,
	}
}

func (c Config) Identity() (identity *age.X25519Identity, err error) {
	if identity, err = age.ParseX25519Identity(c.Privatekey); err != nil {
		log.Fatalf("Could not parse private key: %v", err)
	}
	return identity, nil
}

func (c Config) Recipient() (recipient *age.X25519Recipient, err error) {
	if recipient, err = age.ParseX25519Recipient(c.Publickey); err != nil {
		log.Fatalf("Could not parse public key: %v", err)
	}
	return recipient, err
}

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

type Note struct {
	Uuid  uuid.UUID
	Time  time.Time
	Title string
	Text  string
}

func NewNote(title string, text string) (note *Note) {
	return &Note{
		uuid.New(),
		time.Now(),
		title,
		text,
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
		log.Fatalf("Failed to write encrypted note with uuid %s.", note.Uuid.String())
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

type EncryptedNote struct {
	Uuid       uuid.UUID
	Time       time.Time
	Title      string
	Ciphertext string
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

func (encryptedNote EncryptedNote) ToDecryptedNote(identity age.Identity) (note Note, err error) {
	text, err := encryptedNote.Decrypt(identity)
	return Note{
		encryptedNote.Uuid,
		encryptedNote.Time,
		encryptedNote.Title,
		text,
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
