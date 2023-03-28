package octopus

type Team struct {
	Id                     string
	Name                   string
	MemberUserIds          map[int]string
	ExternalSecurityGroups []string
	CanBeDeleted           bool
	CanBeRenamed           bool
	CanChangeRoles         bool
	CanChangeMembers       bool
	SpaceId                *string
	Slug                   string
	Description            string
}
