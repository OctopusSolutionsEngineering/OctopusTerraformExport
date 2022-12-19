package writers

type writer interface {
	Write(files map[string]string) (string, error)
}
