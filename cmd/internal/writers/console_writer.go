package writers

import "fmt"

type ConsoleWriter struct {
}

func (c ConsoleWriter) Write(files map[string]string) (string, error) {
	for _, v := range files {
		fmt.Println(v)
	}

	return "", nil
}
