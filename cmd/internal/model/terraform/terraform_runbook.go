package terraform

type TerraformRunbook struct {
	Type                     string                          `hcl:"type,label"`
	Name                     string                          `hcl:"name,label"`
	Count                    *string                         `hcl:"count"`
	ResourceName             string                          `hcl:"name"`
	ProjectId                string                          `hcl:"project_id"`
	EnvironmentScope         *string                         `hcl:"environment_scope"`
	Environments             []string                        `hcl:"environments"`
	ForcePackageDownload     bool                            `hcl:"force_package_download "`
	DefaultGuidedFailureMode *string                         `hcl:"default_guided_failure_mode"`
	Description              *string                         `hcl:"description"`
	MultiTenancyMode         *string                         `hcl:"multi_tenancy_mode"`
	RetentionPolicy          *RetentionPolicy                `hcl:"retention_policy,block"`
	ConnectivityPolicy       *TerraformConnectivityPolicy    `hcl:"connectivity_policy,block"`
	Lifecycle                *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}

type RetentionPolicy struct {
	QuantityToKeep    int  `hcl:"quantity_to_keep"`
	ShouldKeepForever bool `hcl:"should_keep_forever"`
}
