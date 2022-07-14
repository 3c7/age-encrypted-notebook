package model

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	Uuid        uuid.UUID
	Time        time.Time
	Title       string
	Text        string
	Attachments []Attachment
}

type Attachment struct {
	Filename string
	Md5      string
	Sha1     string
	Sha256   string
	Sha512   string
	Content  []byte
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
		[]Attachment{},
	}
}

func NewAttachment(filename string, data []byte) (attachment Attachment) {
	bMd5Hash := md5.Sum(data)
	bSha1Hash := sha1.Sum(data)
	bSha256Hash := sha256.Sum256(data)
	bSha512Hash := sha512.Sum512(data)
	return Attachment{
		Filename: filename,
		Md5:      hex.EncodeToString(bMd5Hash[:]),
		Sha1:     hex.EncodeToString(bSha1Hash[:]),
		Sha256:   hex.EncodeToString(bSha256Hash[:]),
		Sha512:   hex.EncodeToString(bSha512Hash[:]),
		Content:  data,
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

func (note *Note) AttachFile(filepath string) (err error) {
	if _, err := os.Stat(filepath); err != nil {
		return err
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	note.Attachments = append(note.Attachments, NewAttachment(gopath.Base(filepath), data))
	return nil
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

func (note *Note) Encrypt(x25519recipients ...age.X25519Recipient) (ciphertext string, encryptedAttachments []EncryptedAttachment, err error) {
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
	ciphertext = base64.StdEncoding.EncodeToString(out.Bytes())
	for i := range note.Attachments {
		currentAttachment := note.Attachments[i]
		out = &bytes.Buffer{}
		w, err = age.Encrypt(out, recipients...)
		if err != nil {
			return "", nil, err
		}

		if err != nil {
			return "", nil, err
		}
		if _, err := w.Write(currentAttachment.Content); err != nil {
			return "", nil, fmt.Errorf("could not write data of attachment %d: %v", i, err)
		}
		if err = w.Close(); err != nil {
			return "", nil, fmt.Errorf("could not close writer after encrypting attachment %d: %v", i, err)
		}
		encryptedAttachments = append(encryptedAttachments, EncryptedAttachment{
			Filename:   currentAttachment.Filename,
			Md5:        currentAttachment.Md5,
			Sha1:       currentAttachment.Sha1,
			Sha256:     currentAttachment.Sha256,
			Sha512:     currentAttachment.Sha512,
			Ciphertext: base64.StdEncoding.EncodeToString(out.Bytes()),
		})
	}
	return ciphertext, encryptedAttachments, nil
}

func (note *Note) ToEncryptedNote(x25519recipients ...age.X25519Recipient) (encryptedNote EncryptedNote, err error) {
	ciphertext, attachments, err := note.Encrypt(x25519recipients...)
	return EncryptedNote{
		Uuid:        note.Uuid,
		Time:        note.Time,
		Title:       note.Title,
		Ciphertext:  ciphertext,
		IsFile:      false,
		Tags:        []string{},
		Attachments: attachments,
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
		Uuid:       bNote.Uuid,
		Time:       bNote.Time,
		Title:      bNote.Title,
		Ciphertext: ciphertext,
		IsFile:     true,
		Tags:       []string{},
	}, err
}

type EncryptedNote struct {
	Uuid        uuid.UUID
	Time        time.Time
	Title       string
	Ciphertext  string
	IsBinary    bool // deprecated
	IsFile      bool
	Tags        []string
	Attachments []EncryptedAttachment
}

type EncryptedAttachment struct {
	Filename   string
	Md5        string
	Sha1       string
	Sha256     string
	Sha512     string
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

// In order to get rid of the "IsBinary" attribute this function can be used to read FileNotes from older databases
func (encryptedNote *EncryptedNote) ContainsFile() bool {
	return encryptedNote.IsBinary || encryptedNote.IsFile
}

// Decrypt decrypts a notes text. For decrypting one of the possible attachments, DecryptAttachment must be called.
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
	if n, err := buffer.ReadFrom(r); err != nil {
		return "", fmt.Errorf("error while after %d bytes from decryption reader: %v", n, err)
	}
	text = buffer.String()
	return text, nil
}

func (encryptedNote *EncryptedNote) DecryptAttachment(num int, identity age.Identity) (attachment Attachment, err error) {
	if num > len(encryptedNote.Attachments) {
		return Attachment{}, errors.New("attachment index out of range.")
	}

	encryptedAttachment := encryptedNote.Attachments[num]
	decoded, err := base64.StdEncoding.DecodeString(encryptedAttachment.Ciphertext)
	if err != nil {
		return Attachment{}, fmt.Errorf("error decoding attachment: %v", err)
	}
	if len(decoded) == 0 && len(encryptedAttachment.Ciphertext) > 0 {
		return Attachment{}, errors.New("decoded attachment is empty, but shouldn't")
	}

	r, err := age.Decrypt(bytes.NewReader(decoded), identity)
	if err != nil {
		return Attachment{}, fmt.Errorf("error decrypting attachment: %v", err)
	}
	buffer := &bytes.Buffer{}
	if n, err := buffer.ReadFrom(r); err != nil {
		return Attachment{}, fmt.Errorf("error after reading %d bytes from decryption reader of attachment: %v", n, err)
	}
	return Attachment{
		Filename: encryptedAttachment.Filename,
		Md5:      encryptedAttachment.Md5,
		Sha1:     encryptedAttachment.Sha1,
		Sha256:   encryptedAttachment.Sha256,
		Sha512:   encryptedAttachment.Sha512,
		Content:  buffer.Bytes(),
	}, nil
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
		nil,
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
			nil,
		},
		content,
	}, err
}

func (encryptedNote *EncryptedNote) Flags() (flags string) {
	if encryptedNote.ContainsFile() {
		flags += "F"
	} else if len(encryptedNote.Attachments) > 0 {
		flags += fmt.Sprintf("A%d", len(encryptedNote.Attachments))
	} else {
		flags += "--"
	}

	if len(encryptedNote.Tags) > 0 {
		flags += "T"
	} else {
		flags += "-"
	}
	return flags
}

func (encryptedNote *EncryptedNote) AddTag(t string) {
	encryptedNote.Tags = append(encryptedNote.Tags, t)
}

func (encryptedNote *EncryptedNote) RemoveTag(t string) error {
	for idx := range encryptedNote.Tags {
		if encryptedNote.Tags[idx] == t {
			tags := encryptedNote.Tags[:idx]
			tags = append(tags, encryptedNote.Tags[idx+1:]...)
			encryptedNote.Tags = tags
			return nil
		}
	}
	return errors.New("tag not found")
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
