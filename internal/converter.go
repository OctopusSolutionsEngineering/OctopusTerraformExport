package internal

type converter interface {
	convertAll() (string, error)
	convertById(id string) (string, error)
	convertByName(name string) (string, error)
	getResourceType() string
}
