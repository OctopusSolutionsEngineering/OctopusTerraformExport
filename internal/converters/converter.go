package converters

type converter interface {
	ToHcl() (map[string]string, error)
	ToHclById(id string) (map[string]string, error)
	ToHclByName(name string) (map[string]string, error)
	GetResourceType() string
}
