package converters

type converter interface {
	ToHcl() (string, error)
	ToHclById(id string) (string, error)
	ToHclByName(name string) (string, error)
	GetResourceType() string
}
