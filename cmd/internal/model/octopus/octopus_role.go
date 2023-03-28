package octopus

type Role struct {
	Id                           string
	Name                         string
	Description                  string
	SupportedRestrictions        *string
	SpacePermissionDescriptions  map[int]string
	SystemPermissionDescriptions []string
	GrantedSpacePermissions      map[int]string
	GrantedSystemPermissions     []string
	CanBeDeleted                 bool
}
