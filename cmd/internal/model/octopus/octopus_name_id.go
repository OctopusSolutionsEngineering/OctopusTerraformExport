package octopus

// NamedResource provides a common interface for any resource that has a name and an ID.
// This is almost every resource in Octopus Deploy.
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
