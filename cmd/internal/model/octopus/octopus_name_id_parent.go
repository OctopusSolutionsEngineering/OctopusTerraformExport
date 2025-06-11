package octopus

// NameIdParent provides a common interface for any resource that has a name, an ID, and an optional parent ID.
type NameIdParent interface {
	NamedResource
	GetParentId() *string
}
