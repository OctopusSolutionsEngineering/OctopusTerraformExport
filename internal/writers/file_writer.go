package writers

import (
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

var tempDir = "/tmp/" + uuid.New().String() + "/"

type FileWriter struct {
}

func (c FileWriter) Write(files map[string]string) (string, error) {
	for k, v := range files {
		if err := c.write(k, v); err != nil {
			return "", err
		}
	}
	return tempDir, nil
}

func (c FileWriter) write(filename string, contents string) error {
	// create the directory
	if err := os.MkdirAll(filepath.Dir(tempDir+filename), os.ModePerm); err != nil {
		return nil
	}

	// create the file
	f, err := os.Create(tempDir + filename)

	if err != nil {
		return err
	}

	defer f.Close()

	f.Write([]byte(contents))

	return nil
}
