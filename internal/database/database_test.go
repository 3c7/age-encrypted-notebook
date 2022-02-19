package database_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/3c7/aen/internal/database"
	"github.com/3c7/aen/internal/model"
	"github.com/google/uuid"
)

// const pub string = "age1z4w8mwlunrg5kx4cjaw2q7kp977vr3edm4wsnutucgjlafy3deyqdeu52k"
// const key string = "AGE-SECRET-KEY-1PXSVSD9FMPFMTD6YMYUP0VLJFURMSE7WF2GQKR73VFN5JZ4CCV3QJFJG54"

func TestDBCreation(t *testing.T) {
	file, err := ioutil.TempFile("", "notes.*.db")
	if err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	DB := database.NewDatabaseInstance(file.Name())
	if err := DB.Open(); err != nil {
		t.Errorf("Could not open database: %v", err)
	}
	defer DB.Close()
}

func TestOpenDBMultiple(t *testing.T) {
	file, err := ioutil.TempFile("", "notes.*.db")
	if err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	DB := database.NewDatabaseInstance(file.Name())
	if err := DB.Open(); err != nil {
		t.Errorf("Could not open database: %v", err)
	}
	defer DB.Close()

	if err := DB.Open(); err == nil {
		t.Error("Call to DB.Open has not returned an error, but should, as the database is already opened.")
	}
	defer DB.Close()
}

func TestWriteEncryptedNote(t *testing.T) {
	file, err := ioutil.TempFile("", "notes.*.db")
	if err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	DB := database.NewDatabaseInstance(file.Name())
	if err := DB.Open(); err != nil {
		t.Errorf("Could not open database: %v", err)
	}
	defer DB.Close()

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

	err = DB.SaveEncryptedNote(&note)
	if err != nil {
		t.Errorf("Could not save note: %v", err)
	}
}

// Tests for multiple recipients in database
func TestMultipleRecipients(t *testing.T) {
	file, err := ioutil.TempFile("", "notes.*.db")
	if err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	DB := database.NewDatabaseInstance(file.Name())
	if err := DB.Open(); err != nil {
		t.Errorf("Could not open database: %v", err)
	}
	defer DB.Close()

	i1, _ := age.GenerateX25519Identity()
	i2, err := age.GenerateX25519Identity()
	if err != nil {
		t.Errorf("Error during identity generation: %v", err)
	}
	t.Logf("Got recipient %s", i1.Recipient().String())
	t.Logf("Got recipient %s", i2.Recipient().String())

	r1 := model.Recipient{
		Alias:     "Test1",
		Publickey: i1.Recipient().String(),
	}
	r2 := model.Recipient{
		Alias:     "Test2",
		Publickey: i2.Recipient().String(),
	}

	_ = DB.AddRecipient(r1)
	err = DB.AddRecipient(r2)
	if err != nil {
		t.Errorf("Error during recipient addition: %v", err)
	}

	r, err := DB.GetRecipients()
	if err != nil {
		t.Errorf("Error loading recipients: %v", err)
	}
	if len(r) != 2 {
		t.Errorf("Length should be 2 but was %d.", len(r))
	}
	for _, r3 := range r {
		t.Logf("Loaded recipient %s", r3.Publickey)
		if r3.Publickey != i1.Recipient().String() && r3.Publickey != i2.Recipient().String() {
			t.Errorf(
				"r3 should be either %s or %s but was %s.",
				i1.Recipient().String(),
				i2.Recipient().String(),
				r3.Publickey,
			)
		}
	}
}
