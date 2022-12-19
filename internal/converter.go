package internal

type converter interface {
	ToHclById(id string) (string, error)
	ToHclByName(name string) (string, error)
	GetResourceType() string
}
