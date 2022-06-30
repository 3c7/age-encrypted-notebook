package model_test

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/3c7/aen/internal/model"
	"github.com/google/uuid"
)

const pub string = "age1z4w8mwlunrg5kx4cjaw2q7kp977vr3edm4wsnutucgjlafy3deyqdeu52k"
const key string = "AGE-SECRET-KEY-1PXSVSD9FMPFMTD6YMYUP0VLJFURMSE7WF2GQKR73VFN5JZ4CCV3QJFJG54"

const pub2 string = "age143en4q09pkgy0ph76uvfkhh656cmsduprmg93kzvynghdzmfqqpqk68avg"
const key2 string = "AGE-SECRET-KEY-1WD0AVC0QCM5XNKJZF7SGGSCACTUQ5QF4TNMWS4UR39ZMS7URNXCQLEG2EP"

func TestNoteCreation(t *testing.T) {
	title := "Read this"
	text := "Hello World!"
	note := model.NewNote(title, text)
	if title != note.Title {
		t.Error("Note title is not as expected.")
	}
	if text != note.Text {
		t.Error("Note text is not as expected.")
	}
}

func TestNoteEncryptionAndDecryption(t *testing.T) {
	const message string = "Testing note encryption!"

	recipient, err := age.ParseX25519Recipient(pub)
	if err != nil {
		t.Errorf("Could not parse public key: %v", err)
	}

	note := model.NewNote("Test", "Testing note encryption!")
	noteJson, err := note.Json()
	if err != nil {
		t.Logf(">>> DEBUG: Could not get json string of note: %v", err)
	} else {
		t.Logf(">>> DEBUG: %s\n", noteJson)
	}
	encryptedNote, err := note.ToEncryptedNote(*recipient)
	if err != nil {
		t.Errorf("Could not encrypt note: %v", err)
	}
	noteJson, err = encryptedNote.Json()
	if err != nil {
		t.Logf(">>> DEBUG: Could not get json string of note: %v", err)
	} else {
		t.Logf(">>> DEBUG: %s\n", noteJson)
	}

	identity, err := age.ParseX25519Identity(key)
	if err != nil {
		t.Errorf("Could not parse private key: %v", err)
	}

	exp, err := encryptedNote.Decrypt(identity)
	if err != nil {
		t.Errorf("Could not decrypt note: %v", err)
	}

	if message != exp {
		t.Errorf("Decrypted message should be \"%s\" but was \"%s\"", message, exp)
	}
}

func TestSlugCreation(t *testing.T) {
	title := "!\"§$%&/()=?Hello World!"
	expected := "hello-world"
	note := model.EncryptedNote{
		Title: title,
	}
	if note.Slug() != expected {
		t.Errorf("Slug mismatch: %s <> %s (current <> exp)", note.Slug(), expected)
	}
}

func TestJsonMarshallNotes(t *testing.T) {
	id, err := uuid.Parse("12345678-1234-5678-9012-123456789012")
	if err != nil {
		t.Errorf("Error parsing UUID: %v", err)
	}
	note := model.EncryptedNote{
		Uuid:       id,
		Time:       time.Now(),
		Title:      "This is my Title!",
		Ciphertext: "Imagine some base64 encoded ciphertext here.",
	}
	jsonNote, err := json.Marshal(note)
	if err != nil {
		t.Errorf("Error during marshalling encrypted note to JSON: %v", err)
	}
	t.Logf(">>> DEBUG - TestJsonMarshallNotes: %s\n", string(jsonNote))
}

func TestRecipientFromKey(t *testing.T) {
	r1 := model.NewRecipient("r1", pub)
	identity, err := age.ParseX25519Identity(key)
	if err != nil {
		t.Errorf("Could not parse identity: %v", err)
	}
	r2 := model.NewRecipientFromIdentity("r2", *identity)

	if r1.Publickey != r2.Publickey {
		t.Errorf("Public keys differ, but should be the same.")
	}
}

func TestEncryptionWithTwoRecipients(t *testing.T) {
	content := "Content"
	i1, err := age.ParseX25519Identity(key)
	if err != nil {
		t.Fatalf("could not parse private key: %v", err)
	}
	i2, err := age.ParseX25519Identity(key2)
	if err != nil {
		t.Fatalf("could not parse private key (2): %v", err)
	}
	note := model.NewNote("Title", content)
	encryptedNote, err := note.ToEncryptedNote(*i1.Recipient(), *i2.Recipient())
	if err != nil {
		t.Fatalf("could not encrypt note: %v", err)
	}

	noteText1, err := encryptedNote.Decrypt(i1)
	if err != nil {
		t.Fatalf("could not decrypt note: %v", err)
	}
	if noteText1 != content {
		t.Fatalf("decrypted content should be %s but was %s", content, noteText1)
	}
	noteText2, err := encryptedNote.Decrypt(i2)
	if err != nil {
		t.Fatalf("could not decrypt note (2): %v", err)
	}
	if noteText2 != content {
		t.Fatalf("decrypted content (2) should be %s but was %s", content, noteText2)
	}
}

func TestBinaryNoteCreation(t *testing.T) {
	content := []byte("This is a string, but it could be any data.")
	binNote := model.NewFileNote("Binary Note Title", content)

	for i := range content {
		if content[i] != binNote.Content[i] {
			t.Fatalf("note content differs at position %d", i)
		}
	}
}

// TestBinaryNoteSlug is basically redundant as the Slug functions is from the Note struct.
func TestBinaryNoteSlug(t *testing.T) {
	content := []byte("This is a string, but it could be any data.")
	slug := "binary-note-title"
	binNote := model.NewFileNote("Binary Note Title", content)

	if binNote.Slug() != slug {
		t.Fatalf("note slug should be %s but was %s", slug, binNote.Slug())
	}
}

func TestBinaryNoteEncryption(t *testing.T) {
	data := []byte{0xc0, 0xde, 0xc0, 0xff, 0xee}
	bNote := model.NewFileNote("My Binary Note", data)
	r1, err := age.ParseX25519Recipient(pub)
	r2, err := age.ParseX25519Recipient(pub2)
	i1, err := age.ParseX25519Identity(key)
	recipients := []age.X25519Recipient{*r1, *r2}
	if err != nil {
		t.Fatal("error parsing recipients or identities")
	}

	encryptedNote, err := bNote.ToEncryptedNote(recipients...)
	if err != nil {
		t.Fatalf("could not encrypt note: %v", err)
	}

	t.Logf(">>> DEBUG encrypted content is %s.", encryptedNote.Ciphertext)

	_, err = encryptedNote.ToDecryptedNote(i1)
	if err == nil {
		t.Fatalf("function call ToDecryptNote should return an error but was nil")
	}

	decryptedNote, err := encryptedNote.ToDecryptedFileNote(i1)
	if err != nil {
		t.Fatalf("could not decrypt note: %v", err)
	}
	for i := range decryptedNote.Content {
		if decryptedNote.Content[i] != data[i] {
			t.Fatalf("note content differs at position %d", i)
		}
	}
}

func TestBinaryNoteFromFile(t *testing.T) {
	content := []byte("This is the content!")
	tmp := t.TempDir()
	err := os.WriteFile(path.Join(tmp, "content"), content, 0600)
	if err != nil {
		t.Fatalf("error writing temporary file: %v", err)
	}

	bNote, err := model.FileNoteFromFile(path.Join(tmp, "content"), "")
	if err != nil {
		t.Fatalf("error creating binary note from file: %v", err)
	}

	if bNote.Title != "content" {
		t.Fatalf("note title is %s but should be content", bNote.Title)
	}
	for i := range bNote.Content {
		if bNote.Content[i] != content[i] {
			t.Fatalf("content differs at position %d", i)
		}
	}
}

func TestTags(t *testing.T) {
	en := model.EncryptedNote{
		Tags: []string{},
	}
	en.AddTag("First")
	en.AddTag("Second")
	en.AddTag("Third")
	err := en.RemoveTag("Second")
	if err != nil {
		t.Fatalf("Error removing tag: %v", err)
	}

	s := strings.Join(en.Tags, "-")
	if s != "First-Third" {
		t.Fatalf("Concatenated string should be First-Third, but was %s", s)
	}
}
