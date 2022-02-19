package utils_test

import (
	"os"
	"strings"
	"testing"

	"github.com/3c7/aen/internal/utils"
)

func TestOverwriteFileContents(t *testing.T) {
	tmp, err := os.CreateTemp("", "overwrite")
	if err != nil {
		t.Errorf("Could not create temporary file: %v", err)
	}

	err = os.WriteFile(tmp.Name(), []byte("Test"), 0600)
	if err != nil {
		t.Errorf("Could not write to temporary file: %v", err)
	}

	err = tmp.Close()
	if err != nil {
		t.Errorf("Could not close temporary file: %v", err)
	}

	err = utils.OverwriteFileContent(tmp.Name())
	if err != nil {
		t.Errorf("Could not write random data to file: %v", err)
	}

	content, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Errorf("Could not read from temporary file: %v", err)
	}

	if strings.Contains(string(content), "Test") {
		t.Errorf("Content of file was not overwritten.")
	}
	t.Logf("File content seems to be overwritten: %s", string(content))
	os.Remove(tmp.Name())
}
