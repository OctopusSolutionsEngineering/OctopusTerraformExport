package terraform

// https://registry.terraform.io/providers/scottwinkler/shell/latest/docs/resources/shell_script_resource
type TerraformShellScript struct {
	Type                 string                                `hcl:"type,label"`
	Name                 string                                `hcl:"name,label"`
	Count                *string                               `hcl:"count"`
	LifecycleCommands    TerraformShellScriptLifecycleCommands `hcl:"lifecycle_commands,block"`
	Environment          map[string]string                     `hcl:"environment"`
	SensitiveEnvironment map[string]string                     `hcl:"sensitive_environment"`
	WorkingDirectory     *string                               `hcl:"working_directory"`
}

type TerraformShellScriptLifecycleCommands struct {
	Read   *string `hcl:"read"`
	Create string  `hcl:"create"`
	Update *string `hcl:"update"`
	Delete string  `hcl:"delete"`
}
