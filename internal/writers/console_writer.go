package writers

import "fmt"

type ConsoleWriter struct {
}

func (c ConsoleWriter) Write(files map[string]string) (string, error) {
	for k, v := range files {
		fmt.Println(k)
		fmt.Println(v)
	}

	return "", nil
}
