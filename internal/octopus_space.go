package internal

type Space struct {
	Id                       string
	Name                     string
	Description              string
	IsDefault                bool
	TaskQueueStopped         bool
	SpaceManagersTeams       []string
	SpaceManagersTeamMembers []string
}

type SpaceCollection struct {
	Items          []Space
	TotalResults   int
	ItemsPerPage   int
	NumberOfPages  int
	LastPageNumber int
	ItemType       string
}
