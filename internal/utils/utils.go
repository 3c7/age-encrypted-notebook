package utils

import (
	"crypto/rand"
	"os"
	"regexp"

	"filippo.io/age"
)

// Loads private key file, strips content (e.g. additional linebreaks) and passes it to age.ParseX25519Identity.
func IdentityFromKeyfile(path string) (identity *age.X25519Identity, err error) {
	if _, err = os.Stat(path); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	regex, err := regexp.Compile("[^a-zA-Z0-9-]")
	if err != nil {
		return nil, err
	}
	keyString := regex.ReplaceAllString(string(content), "")
	return age.ParseX25519Identity(keyString)
}

// Overwrites file content with generated pseudo random numbers
func OverwriteFileContent(path string) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b := make([]byte, info.Size())
	_, err = rand.Read(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
