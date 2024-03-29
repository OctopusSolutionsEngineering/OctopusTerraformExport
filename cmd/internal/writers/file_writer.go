package writers

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

type FileWriter struct {
	dest string
}

func NewFileWriterToTempDir() *FileWriter {
	return &FileWriter{
		dest: os.TempDir() + string(os.PathSeparator) + uuid.New().String() + string(os.PathSeparator),
	}
}

func NewFileWriter(dest string) *FileWriter {
	if dest == "" {
		return NewFileWriterToTempDir()
	}

	return &FileWriter{
		dest: strutil.EnsureSuffix(dest, string(os.PathSeparator)),
	}
}

func (c FileWriter) Write(files map[string]string) (string, error) {
	for k, v := range files {
		if err := c.write(k, v); err != nil {
			return "", err
		}
	}
	return c.dest, nil
}

func (c FileWriter) write(filename string, contents string) (funcErr error) {
	// create the directory
	if err := os.MkdirAll(filepath.Dir(c.dest+filename), os.ModePerm); err != nil {
		return err
	}

	// create the file
	f, err := os.Create(c.dest + filename)
	if err != nil {
		return err
	}

	defer func(f *os.File) {
		deferErr := f.Close()
		if deferErr != nil && err == nil {
			funcErr = deferErr
		}
	}(f)

	_, err = f.Write([]byte(contents))
	if err != nil {
		return err
	}

	return nil
}
