package octopus

type Role struct {
	Id                           string
	Name                         string
	Description                  string
	SupportedRestrictions        *string
	SpacePermissionDescriptions  map[int]string
	SystemPermissionDescriptions map[int]string
	GrantedSpacePermissions      map[int]string
	GrantedSystemPermissions     map[int]string
	CanBeDeleted                 bool
}
