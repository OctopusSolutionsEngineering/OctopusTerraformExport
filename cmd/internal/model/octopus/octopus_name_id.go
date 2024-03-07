package octopus

type NamedResource interface {
	GetName() string
}

type NameId struct {
	Id   string
	Name string
}

func (n NameId) GetName() string {
	return n.Name
}
