package output

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/writers"
)

func WriteFiles(files map[string]string, dest string, console bool) error {
	if dest != "" {
		writer := writers.NewFileWriter(dest)
		_, err := writer.Write(files)
		if err != nil {
			return err
		}
	}

	if console || dest == "" {
		consoleWriter := writers.ConsoleWriter{}
		output, err := consoleWriter.Write(files)
		if err != nil {
			return err
		}
		fmt.Println(output)
	}

	return nil
}
