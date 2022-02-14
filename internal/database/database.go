package database

import (
	"encoding/json"
	"errors"
	"fmt"

	"filippo.io/age"
	"github.com/3c7/aen/internal/model"
	bolt "go.etcd.io/bbolt"
)

type Database struct {
	Path   string
	Handle *bolt.DB
	isOpen bool
}

func NewDatabaseInstance(path string) *Database {
	return &Database{
		Path:   path,
		isOpen: false,
	}
}

func (db *Database) Open() (err error) {
	if db.isOpen {
		return errors.New("Database is already open")
	}
	db.Handle, err = bolt.Open(db.Path, 0600, nil)
	if err == nil {
		db.isOpen = true
	}
	return err
}

func (db *Database) Close() (err error) {
	err = db.Handle.Close()
	if err == nil {
		db.isOpen = false
	}
	return err
}

func (db *Database) ensureBucket(tx *bolt.Tx, bucket []byte) (b *bolt.Bucket, err error) {
	if tx.DB().IsReadOnly() {
		return nil, errors.New("database is read-only")
	}
	b = tx.Bucket(bucket)
	if b == nil {
		return tx.CreateBucket(bucket)
	}
	return b, nil
}

func (db *Database) writeToBucket(tx *bolt.Tx, bucket []byte, key []byte, value []byte) (err error) {
	if !db.isOpen {
		return errors.New("database is not open")
	}
	b, err := db.ensureBucket(tx, bucket)
	if err != nil {
		return err
	}
	return b.Put(key, value)
}

func (db *Database) readFromBucket(tx *bolt.Tx, bucket []byte, key []byte) (value []byte, err error) {
	if !db.isOpen {
		return nil, errors.New("database is not open")
	}
	b, err := db.ensureBucket(tx, bucket)
	if err != nil {
		return nil, err
	}
	return b.Get(key), nil
}

func (db *Database) deleteFromBucket(tx *bolt.Tx, bucket []byte, key []byte) (err error) {
	if !db.isOpen {
		return errors.New("database is not open")
	}
	b, err := db.ensureBucket(tx, bucket)
	if err != nil {
		return err
	}
	return b.Delete(key)
}

func (db *Database) SaveEncryptedNote(encryptedNote *model.EncryptedNote) (err error) {
	return db.Handle.Update(func(tx *bolt.Tx) error {
		buf, err := json.Marshal(encryptedNote)
		if err != nil {
			return err
		}
		slug := encryptedNote.Slug()
		return db.writeToBucket(tx, []byte("notes"), []byte(slug), buf)
	})
}

func (db *Database) GetEncryptedNotes() (notes []model.EncryptedNote, err error) {
	err = db.Handle.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("notes"))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			var note model.EncryptedNote
			err := json.Unmarshal(v, &note)
			if err != nil {
				return err
			}
			notes = append(notes, note)
			return nil
		})
		return nil
	})
	return notes, err
}

func (db *Database) GetEncryptedNoteBySlug(slug string) (encryptedNote *model.EncryptedNote, err error) {
	var note model.EncryptedNote
	err = db.Handle.View(func(tx *bolt.Tx) error {
		buf, err := db.readFromBucket(tx, []byte("notes"), []byte(slug))
		if err != nil {
			return err
		}
		err = json.Unmarshal(buf, &note)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not get encrypted note from database: %v", err)
	}
	return &note, nil
}

func (db *Database) DeleteNoteBySlug(slug string) (err error) {
	err = db.Handle.Update(func(tx *bolt.Tx) error {
		err := db.deleteFromBucket(tx, []byte("notes"), []byte(slug))
		return err
	})
	return err
}

func (db *Database) GetEncryptedNoteByIndex(idx int) (encryptedNote *model.EncryptedNote, err error) {
	notes, err := db.GetEncryptedNotes()
	if err != nil {
		return nil, err
	}

	if len(notes)+1 <= idx {
		return nil, errors.New("index is out of range")
	}

	model.SortNoteSlice(notes)
	return &notes[idx-1], nil
}

func (db *Database) GetReceipients() (recipients []age.X25519Recipient, err error) {
	// Using read/write access because the Bucket might not exist yet
	err = db.Handle.Update(func(tx *bolt.Tx) error {
		var publicKeys []string
		buf, err := db.readFromBucket(tx, []byte("config"), []byte("recipients"))
		if err != nil {
			return err
		}
		if len(string(buf)) > 0 {
			err = json.Unmarshal(buf, &publicKeys)
			if err != nil {
				return err
			}
			for _, key := range publicKeys {
				recipient, err := age.ParseX25519Recipient(key)
				if err != nil {
					return err
				}
				recipients = append(recipients, *recipient)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

func (db *Database) AddRecipient(r *age.X25519Recipient) (err error) {
	var (
		recipients []age.X25519Recipient
		publicKeys []string
	)
	recipients, err = db.GetReceipients()
	if err != nil {
		return err
	}

	for _, recipient := range recipients {
		if (*r).String() == recipient.String() {
			return nil
		}
		publicKeys = append(publicKeys, recipient.String())
	}
	publicKeys = append(publicKeys, r.String())

	buf, err := json.Marshal(publicKeys)
	if err != nil {
		return err
	}
	err = db.Handle.Update(func(tx *bolt.Tx) error {
		return db.writeToBucket(tx, []byte("config"), []byte("recipients"), buf)
	})
	return err
}