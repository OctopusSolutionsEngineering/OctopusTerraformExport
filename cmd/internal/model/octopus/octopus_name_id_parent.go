package octopus

// NameIdParentResource provides a common interface for any resource that has a name, an ID, and an optional parent ID.
type NameIdParentResource interface {
	NamedResource
	GetParentId() *string
	GetUltimateParent() string
}
