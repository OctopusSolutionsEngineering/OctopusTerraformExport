package terraform

type TerraformPlatformHubVersionControlAnonymousSettings struct {
	Type          string  `hcl:"type,label"`
	Name          string  `hcl:"name,label"`
	Count         *string `hcl:"count"`
	Url           string  `hcl:"url"`
	DefaultBranch string  `hcl:"default_branch"`
	BasePath      string  `hcl:"base_path"`
}
