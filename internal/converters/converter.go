package converters

type ConverterID interface {
	ToHclById(id string) (map[string]string, error)
	GetResourceType() string
}

type ConverterName interface {
	ToHclByName(name string) (map[string]string, error)
	GetResourceType() string
}

type ConverterAll interface {
	ToHcl() (map[string]string, error)
	ToHclById(id string) (map[string]string, error)
	ToHclByName(name string) (map[string]string, error)
	GetResourceType() string
}
