package octopus

type OctopusPlatformHubVersionControlUsernamePasswordSetting struct {
	Url           string
	DefaultBranch string
	BasePath      string
	Credentials   OctopusPlatformHubVersionControlUsernamePasswordSettingCredentials
}

type OctopusPlatformHubVersionControlUsernamePasswordSettingCredentials struct {
	Type     string
	Username string
	Password map[string]any
}
