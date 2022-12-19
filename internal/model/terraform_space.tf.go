package model

type TerraformSpace struct {
	Type                     string   `hcl:"type,label"`
	Name                     string   `hcl:"name,label"`
	Description              string   `hcl:"description"`
	ResourceName             string   `hcl:"name"`
	IsDefault                bool     `hcl:"is_default"`
	IsTaskQueueStopped       bool     `hcl:"is_task_queue_stopped"`
	SpaceManagersTeamMembers []string `hcl:"space_managers_team_members"`
	SpaceManagersTeams       []string `hcl:"space_managers_teams"`
}
