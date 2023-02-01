package octopus

type Space struct {
	Id                       string
	Name                     string
	Description              *string
	IsDefault                bool
	TaskQueueStopped         bool
	SpaceManagersTeams       []string
	SpaceManagersTeamMembers []string
}
