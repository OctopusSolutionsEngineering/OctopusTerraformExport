package octopus

type NamedResource interface {
	GetName() string
	GetId() string
}

type NameId struct {
	Id      string
	SpaceId string
	Name    string
}

func (n NameId) GetName() string {
	return n.Name
}

func (n NameId) GetId() string {
	return n.Id
}
