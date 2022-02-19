package model_test

import (
	"encoding/json"
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
