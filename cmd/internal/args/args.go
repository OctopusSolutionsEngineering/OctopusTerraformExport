package args

import (
	"bytes"
	"errors"
	"flag"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type Arguments struct {
	Profiling                   bool
	ExcludeTerraformVariables   bool
	ExcludeSpaceCreation        bool
	ConfigFile                  string
	ConfigPath                  string
	Version                     bool
	IgnoreInvalidExcludeExcept  bool
	Url                         string
	ApiKey                      string
	Space                       string
	Destination                 string
	Console                     bool
	ProjectId                   Projects
	ProjectName                 Projects
	RunbookId                   string
	RunbookName                 string
	LookupProjectDependencies   bool
	Stateless                   bool
	StatelessAdditionalParams   StringSliceArgs
	StepTemplateName            string
	StepTemplateKey             string
	StepTemplateDescription     string
	IgnoreCacManagedValues      bool
	ExcludeCaCProjectSettings   bool
	BackendBlock                string
	DetachProjectTemplates      bool
	DefaultSecretVariableValues bool
	DummySecretVariableValues   bool
	ProviderVersion             string
	ExcludeProvider             bool
	IncludeOctopusOutputVars    bool
	LimitAttributeLength        int
	LimitResourceCount          int
	GenerateImportScripts       bool

	IgnoreProjectChanges         bool
	IgnoreProjectVariableChanges bool
	IgnoreProjectGroupChanges    bool
	IgnoreProjectNameChanges     bool
	LookUpDefaultWorkerPools     bool
	IncludeIds                   bool
	IncludeSpaceInPopulation     bool
	IncludeDefaultChannel        bool

	/*
		We expose a lot of options to exclude resources from the export.

		Some of these exclusions are "natural" in that you can simply not export them and the resulting Terraform configuration will be (mostly) valid.
		For example, you can exclude targets and runbooks and the only impact is that some scoped variables may become unscoped. Or you can exclude
		projects and only "Deploy a Release" steps will be impacted.

		Other exclusions are "core" in that they are tightly coupled to many other resources. For example, excluding environments, lifecycles,
		project groups, accounts etc. will likely leave the resulting Terraform configuration in an invalid state.

		Warnings have been added to the CLI help command to identify some of the side-effects of the various exclusions.
	*/

	ExcludeAllProjectVariables       bool
	ExcludeProjectVariables          StringSliceArgs
	ExcludeProjectVariablesExcept    StringSliceArgs
	ExcludeProjectVariablesRegex     StringSliceArgs
	ExcludeVariableEnvironmentScopes StringSliceArgs

	ExcludeAllTenantVariables    bool
	ExcludeTenantVariables       StringSliceArgs
	ExcludeTenantVariablesExcept StringSliceArgs
	ExcludeTenantVariablesRegex  StringSliceArgs

	ExcludeAllSteps    bool
	ExcludeSteps       StringSliceArgs
	ExcludeStepsRegex  StringSliceArgs
	ExcludeStepsExcept StringSliceArgs

	ExcludeAllRunbooks    bool
	ExcludeRunbooks       StringSliceArgs
	ExcludeRunbooksRegex  StringSliceArgs
	ExcludeRunbooksExcept StringSliceArgs

	ExcludeLibraryVariableSets       StringSliceArgs
	ExcludeLibraryVariableSetsRegex  StringSliceArgs
	ExcludeLibraryVariableSetsExcept StringSliceArgs
	ExcludeAllLibraryVariableSets    bool

	ExcludeEnvironments       StringSliceArgs
	ExcludeEnvironmentsRegex  StringSliceArgs
	ExcludeEnvironmentsExcept StringSliceArgs
	ExcludeAllEnvironments    bool

	ExcludeFeeds       StringSliceArgs
	ExcludeFeedsRegex  StringSliceArgs
	ExcludeFeedsExcept StringSliceArgs
	ExcludeAllFeeds    bool

	ExcludeProjectGroups       StringSliceArgs
	ExcludeProjectGroupsRegex  StringSliceArgs
	ExcludeProjectGroupsExcept StringSliceArgs
	ExcludeAllProjectGroups    bool

	ExcludeAccounts       StringSliceArgs
	ExcludeAccountsRegex  StringSliceArgs
	ExcludeAccountsExcept StringSliceArgs
	ExcludeAllAccounts    bool

	ExcludeCertificates       StringSliceArgs
	ExcludeCertificatesRegex  StringSliceArgs
	ExcludeCertificatesExcept StringSliceArgs
	ExcludeAllCertificates    bool

	ExcludeLifecycles       StringSliceArgs
	ExcludeLifecyclesRegex  StringSliceArgs
	ExcludeLifecyclesExcept StringSliceArgs
	ExcludeAllLifecycles    bool

	ExcludeWorkerpools       StringSliceArgs
	ExcludeWorkerpoolsRegex  StringSliceArgs
	ExcludeWorkerpoolsExcept StringSliceArgs
	ExcludeAllWorkerpools    bool

	ExcludeMachinePolicies       StringSliceArgs
	ExcludeMachinePoliciesRegex  StringSliceArgs
	ExcludeMachinePoliciesExcept StringSliceArgs
	ExcludeAllMachinePolicies    bool

	ExcludeTenantTags StringSliceArgs

	ExcludeTenantTagSets       StringSliceArgs
	ExcludeTenantTagSetsRegex  StringSliceArgs
	ExcludeTenantTagSetsExcept StringSliceArgs
	ExcludeAllTenantTagSets    bool

	ExcludeTenants         StringSliceArgs
	ExcludeTenantsRegex    StringSliceArgs
	ExcludeTenantsWithTags StringSliceArgs
	ExcludeTenantsExcept   StringSliceArgs
	ExcludeAllTenants      bool

	ExcludeProjects       StringSliceArgs
	ExcludeProjectsExcept StringSliceArgs
	ExcludeProjectsRegex  StringSliceArgs
	ExcludeAllProjects    bool

	ExcludeAllTargets                bool
	ExcludeTargets                   StringSliceArgs
	ExcludeTargetsRegex              StringSliceArgs
	ExcludeTargetsExcept             StringSliceArgs
	ExcludeTargetsWithNoEnvironments bool

	ExcludeAllGitCredentials bool
}

// GetBackend forces the use of a local backend for stateless exports
func (arguments *Arguments) GetBackend() string {
	if arguments.Stateless {
		return ""
	}

	return arguments.BackendBlock
}

type Projects []string

func (i *Projects) String() string {
	return "exported projects"
}

func (i *Projects) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type StringSliceArgs []string

func (i *StringSliceArgs) String() string {
	return "A collection of strings passed as arguments"
}

func (i *StringSliceArgs) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

func ParseArgs(args []string) (Arguments, string, error) {
	flags := flag.NewFlagSet("octoterra", flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	arguments := Arguments{}

	flags.StringVar(&arguments.ConfigFile, "configFile", "octoterra", "The name of the configuration file to use. Do not include the extension. Defaults to octoterra")
	flags.StringVar(&arguments.ConfigPath, "configPath", ".", "The path of the configuration file to use. Defaults to the current directory")
	flags.IntVar(&arguments.LimitAttributeLength, "limitAttributeLength", 0, "For internal use only. Limits the length of the attribute names.")
	flags.IntVar(&arguments.LimitResourceCount, "limitResourceCount", 0, "For internal use only. Limits the number of resources of a given type that are returned. For example, a value of 30 will ensure the exported Terraform only includes up to 30 accounts, and up to 30 feeds, and up to 30 projects etc. This is used to reduce the output when octoterra is used to generate a context for an LLM.")
	flags.BoolVar(&arguments.Profiling, "profiling", false, "Enable profiling. Run \"pprof -http=:8080 octoterra.prof\" to view the results.")
	flags.BoolVar(&arguments.ExcludeTerraformVariables, "excludeTerraformVariables", false, "This option means the exported module does not expose Terraform variables for common inputs like the value of project or library variables set variables. This reduces the size of the Terraform configuration files, but makes the module less configurable because values are hard coded.")
	flags.BoolVar(&arguments.ExcludeSpaceCreation, "excludeSpaceCreation", false, "This option excludes the Terraform configuration that is used to create the space.")
	flags.BoolVar(&arguments.IgnoreInvalidExcludeExcept, "ignoreInvalidExcludeExcept", false, "Ensures that resource names passed to the 'Exclude<ResourceType>Except' arguments are valid, and if they are not, removes those names from the list. This is useful when an external system attempts to filter results but places incorrect values into 'Exclude<ResourceType>Except' arguments. It may result in all resources being returned if no valid resources names are included in the 'Exclude<ResourceType>Except' arguments.")
	flags.BoolVar(&arguments.Version, "version", false, "Print the version")
	flags.BoolVar(&arguments.IncludeIds, "includeIds", false, "For internal use only. Include the \"id\" field on generated resources. Note that this is almost always unnecessary and undesirable.")
	flags.BoolVar(&arguments.IncludeSpaceInPopulation, "includeSpaceInPopulation", false, "For internal use only. Include the space resource in the space population script. Note that this is almost always unnecessary and undesirable, as the space resources are included in the space creation module.")
	flags.BoolVar(&arguments.GenerateImportScripts, "generateImportScripts", false, "Generate Bash and Powershell scripts used to import resources into the Terraform state.")
	flags.StringVar(&arguments.Url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")
	flags.StringVar(&arguments.Space, "space", "", "The Octopus space name or ID")
	flags.StringVar(&arguments.ApiKey, "apiKey", "", "The Octopus api key")
	flags.StringVar(&arguments.Destination, "dest", "", "The directory to place the Terraform files in")
	flags.BoolVar(&arguments.Console, "console", false, "Dump Terraform files to the console")
	flags.BoolVar(&arguments.Stateless, "stepTemplate", false, "Create an Octopus step template")
	flags.Var(&arguments.StatelessAdditionalParams, "stepTemplateAdditionalParameters", "Indicates that a non-secret variable should be exposed as a parameter. The format of this option is \"ProjectName:VariableName\". This option is only used with the -stepTemplate option.")
	flags.StringVar(&arguments.StepTemplateName, "stepTemplateName", "", "Step template name. Only used with the stepTemplate option.")
	flags.StringVar(&arguments.StepTemplateKey, "stepTemplateKey", "", "Step template key used when building parameter names. Only used with the stepTemplate option.")
	flags.StringVar(&arguments.StepTemplateDescription, "stepTemplateDescription", "", "Step template description used when building parameter names. Only used with the stepTemplate option.")
	flags.Var(&arguments.ProjectId, "projectId", "Limit the export to a single project")
	flags.Var(&arguments.ProjectName, "projectName", "Limit the export to a single project")
	flags.StringVar(&arguments.RunbookId, "runbookId", "", "Limit the export to a single runbook. Runbooks are exported referencing external resources as data sources.")
	flags.StringVar(&arguments.RunbookName, "runbookName", "", "Limit the export to a single runbook. Requires projectName or projectId. Runbooks are exported referencing external resources as data sources.")
	flags.BoolVar(&arguments.LookupProjectDependencies, "lookupProjectDependencies", false, "Use data sources to lookup the external project dependencies. Use this when the destination space has existing environments, accounts, tenants, feeds, git credentials, and library variable sets that this project should reference.")
	flags.BoolVar(&arguments.IgnoreCacManagedValues, "ignoreCacManagedValues", true, "Pass this to exclude values managed by Config-as-Code from the exported Terraform. This includes non-sensitive variables, the deployment process, connectivity settings, and other project settings. This has no effect on projects that do not have CaC enabled.")
	flags.BoolVar(&arguments.ExcludeCaCProjectSettings, "excludeCaCProjectSettings", false, "Pass this to exclude any Config-As-Code settings in the exported projects. Typically you set -ignoreCacManagedValues=false -excludeCaCProjectSettings=true to essentially \"convert\" a CaC project to a regular project. Values from the \"main\" or \"master\" branches will be used first, or just fall back to the first configured branch.")
	flags.BoolVar(&arguments.DefaultSecretVariableValues, "defaultSecretVariableValues", false, "Pass this to set the default value of secret variables to the octostache template referencing the variable.")
	flags.BoolVar(&arguments.DummySecretVariableValues, "dummySecretVariableValues", false, "Pass this to set the default value of secret variables, account secrets, feed credentials to a dummy value. This allows resources with secret values to be created without knowing the secrets, while still allowing the secret values to be specified if they are known. This option takes precedence over the defaultSecretVariableValues option.")
	flags.StringVar(&arguments.BackendBlock, "terraformBackend", "", "Specifies the backend type to be added to the exported Terraform configuration.")
	flags.StringVar(&arguments.ProviderVersion, "providerVersion", "", "Specifies the Octopus Terraform provider version.")
	flags.BoolVar(&arguments.DetachProjectTemplates, "detachProjectTemplates", false, "Detaches any step templates in the exported Terraform.")
	flags.BoolVar(&arguments.IncludeDefaultChannel, "includeDefaultChannel", false, "Internal use only. Includes the \"Default\" channel as a standard channel resource rather than a data block.")

	flags.BoolVar(&arguments.ExcludeAllSteps, "excludeAllSteps", false, "Exclude all steps when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(&arguments.ExcludeSteps, "excludeSteps", "A steps to be excluded when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(&arguments.ExcludeStepsRegex, "excludeStepsRegex", "A step to be excluded when exporting projects or runbooks based on regex match. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(&arguments.ExcludeStepsExcept, "excludeStepsExcept", "All step except those defined with excludeRunbooksExcept are excluded when exporting a project or runbook. WARNING: variables scoped to other step will no longer have the step scope applied.")

	flags.BoolVar(&arguments.ExcludeAllRunbooks, "excludeAllRunbooks", false, "Exclude all runbooks when exporting a project or space. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(&arguments.ExcludeRunbooks, "excludeRunbook", "A runbook to be excluded when exporting a single project. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(&arguments.ExcludeRunbooksRegex, "excludeRunbookRegex", "A runbook to be excluded when exporting a single project based on regex match. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(&arguments.ExcludeRunbooksExcept, "excludeRunbooksExcept", "All runbooks except those defined with excludeRunbooksExcept are excluded when exporting a single project. WARNING: variables scoped to other runbooks will no longer have the runbook scope applied.")

	flags.BoolVar(&arguments.ExcludeAllLibraryVariableSets, "excludeAllLibraryVariableSets", false, "Exclude all library variable sets. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(&arguments.ExcludeLibraryVariableSets, "excludeLibraryVariableSet", "A library variable set to be excluded when exporting a single project. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(&arguments.ExcludeLibraryVariableSetsRegex, "excludeLibraryVariableSetRegex", "A library variable set to be excluded when exporting a single project based on regex match. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(&arguments.ExcludeLibraryVariableSetsExcept, "excludeLibraryVariableSetsExcept", "All library variable sets except those defined with excludeAllLibraryVariableSets are excluded. WARNING: projects that linked other library variable set will no longer include these variables.")

	flags.BoolVar(&arguments.ExcludeAllEnvironments, "excludeAllEnvironments", false, "Exclude all environments.  WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeEnvironments, "excludeEnvironments", "An environment to be excluded when exporting a single project. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeEnvironmentsRegex, "excludeEnvironmentsRegex", "A environment to be excluded when exporting a single project based on regex match. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeEnvironmentsExcept, "excludeEnvironmentsExcept", "All environments except those defined with excludeEnvironmentsExcept are excluded.  WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllFeeds, "excludeAllFeeds", false, "Exclude all feeds.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeFeeds, "excludeFeeds", "A feed to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeFeedsRegex, "excludeFeedsRegex", "A feed to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeFeedsExcept, "excludeFeedsExcept", "All feeds except those defined with excludeFeedsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllAccounts, "excludeAllAccounts", false, "Exclude all Accounts.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeAccounts, "excludeAccounts", "A feed to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeAccountsRegex, "excludeAccountsRegex", "An account to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeAccountsExcept, "excludeAccountsExcept", "All accounts except those defined with excludeAccountsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllProjectGroups, "excludeAllProjectGroups", false, "Exclude all project groups.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeProjectGroups, "excludeProjectGroups", "A project group to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeProjectGroupsRegex, "excludeProjectGroupsRegex", "A project group to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeProjectGroupsExcept, "excludeProjectGroupsExcept", "All project groups except those defined with excludeProjectGroupsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllCertificates, "excludeAllCertificates", false, "Exclude all Certificates.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeCertificates, "excludeCertificates", "A certificate to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeCertificatesRegex, "excludeCertificatesRegex", "A certificate to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeCertificatesExcept, "excludeCertificatesExcept", "All Certificates except those defined with excludeCertificatesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllLifecycles, "excludeAllLifecycles", false, "Exclude all lifecycles.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeLifecycles, "excludeLifecycles", "A lifecycle to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeLifecyclesRegex, "excludeLifecyclesRegex", "A lifecycle to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeLifecyclesExcept, "excludeLifecyclesExcept", "All lifecycles except those defined with excludeLifecyclesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllWorkerpools, "excludeAllWorkerPools", false, "Exclude all worker pools.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeWorkerpools, "excludeWorkerPools", "A worker pool to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeWorkerpoolsRegex, "excludeWorkerPoolsRegex", "A worker pool to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeWorkerpoolsExcept, "excludeWorkerPoolsExcept", "All worker pools except those defined with excludeWorkerpoolsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllMachinePolicies, "excludeAllMachinePolicies", false, "Exclude all machine policies.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeMachinePolicies, "excludeMachinePolicies", "A machine policy to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeMachinePoliciesRegex, "excludeMachinePoliciesRegex", "A machine policy to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(&arguments.ExcludeMachinePoliciesExcept, "excludeMachinePoliciesExcept", "All machine policies except those defined with excludeMachinePoliciesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(&arguments.ExcludeAllTenantVariables, "excludeAllTenantVariables", false, "Exclude all tenant variables from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(&arguments.ExcludeTenantVariables, "excludeTenantVariables", "Exclude a tenant variable from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(&arguments.ExcludeTenantVariablesExcept, "excludeTenantVariablesExcept", "All tenant variables except those defined with excludeTenantVariablesExcept are excluded. WARNING: steps that used other variables may no longer function correctly.")
	flags.Var(&arguments.ExcludeTenantVariablesRegex, "excludeTenantVariablesRegex", "Exclude a tenant variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly.")

	flags.BoolVar(&arguments.ExcludeAllProjectVariables, "excludeAllProjectVariables", false, "Exclude all project variables from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(&arguments.ExcludeProjectVariables, "excludeProjectVariable", "Exclude a project variable from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(&arguments.ExcludeProjectVariablesExcept, "excludeProjectVariableExcept", "All project variables except those defined with excludeProjectVariableExcept are excluded. WARNING: steps that used other variables may no longer function correctly.")
	flags.Var(&arguments.ExcludeProjectVariablesRegex, "excludeProjectVariableRegex", "Exclude a project variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(&arguments.ExcludeVariableEnvironmentScopes, "excludeVariableEnvironmentScopes", "Exclude a environment when it appears in a variable's environment scope. WARNING: variables scoped to this environment will no longer have that environment scope applied.")

	// missing all, regex, except
	flags.Var(&arguments.ExcludeTenantTags, "excludeTenantTags", "Exclude an individual tenant tag from being exported. Tags are in the format \"taggroup/tagname\". WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied.")

	flags.Var(&arguments.ExcludeTenantTagSets, "excludeTenantTagSets", "Exclude a tenant tag set from being exported. WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied.")
	flags.BoolVar(&arguments.ExcludeAllTenantTagSets, "excludeAllTenantTagSets", false, "Exclude all tenant tag sets from being exported. WARNING: Resources scoped to tenant tag sets will be unscoped.")
	flags.Var(&arguments.ExcludeTenantTagSetsRegex, "excludeTenantTagSetsRegex", "Exclude tenant tag sets from being exported based on a regex. WARNING: Resources scoped to tenant tag sets will be unscoped")
	flags.Var(&arguments.ExcludeTenantTagSetsExcept, "excludeTenantTagSetsExcept", "Exclude all tenant tag sets except for those define in this list. WARNING: Resources scoped to tenant tag sets will be unscoped")

	flags.BoolVar(&arguments.ExcludeAllTenants, "excludeAllTenants", false, "Exclude all tenants from being exported.")
	flags.Var(&arguments.ExcludeTenants, "excludeTenants", "Exclude a tenant from being exported.")
	flags.Var(&arguments.ExcludeTenantsRegex, "excludeTenantsRegex", "Exclude a tenant from being exported based on a regex.")
	flags.Var(&arguments.ExcludeTenantsWithTags, "excludeTenantsWithTag", "Exclude any tenant with this tag from being exported. This is useful when using tags to separate tenants that can be exported with those that should not. Tags are in the format TagGroupName/TagName.")
	flags.Var(&arguments.ExcludeTenantsExcept, "excludeTenantsExcept", "Exclude all tenants except for those define in this list. The tenants in excludeTenants take precedence, so a tenant define here and in excludeTenants is excluded.")

	flags.BoolVar(&arguments.ExcludeAllTargets, "excludeAllTargets", false, "Exclude all targets from being exported. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.BoolVar(&arguments.ExcludeTargetsWithNoEnvironments, "excludeTargetsWithNoEnvironments", false, "Exclude targets that have had all their environments excluded. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(&arguments.ExcludeTargets, "excludeTargets", "Exclude targets from being exported. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(&arguments.ExcludeTargetsRegex, "excludeTargetsRegex", "Exclude targets from being exported based on a regex. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(&arguments.ExcludeTargetsExcept, "excludeTargetsExcept", "Exclude all targets except for those define in this list. The targets in excludeTargets take precedence, so a tenant define here and in excludeTargets is excluded. WARNING: Variables that were scoped to other targets will become unscoped.")

	flags.BoolVar(&arguments.ExcludeAllProjects, "excludeAllProjects", false, "Exclude all projects from being exported. This is only used when exporting a space.")
	flags.Var(&arguments.ExcludeProjects, "excludeProjects", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.Var(&arguments.ExcludeProjectsRegex, "excludeProjectsRegex", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.Var(&arguments.ExcludeProjectsExcept, "excludeProjectsExcept", "All projects except those defined with excludeProjectsExcept are excluded. This is only used when exporting a space.")

	flags.BoolVar(&arguments.ExcludeAllGitCredentials, "excludeAllGitCredentials", false, "Exclude all git credentials. Must be used with -excludeCaCProjectSettings.")

	flags.BoolVar(&arguments.ExcludeProvider, "excludeProvider", false, "Exclude the provider from the exported Terraform configuration files. This is useful when you want to use a parent module to define the backend, as the parent module must define the provider.")
	flags.BoolVar(&arguments.IncludeOctopusOutputVars, "includeOctopusOutputVars", true, "Capture the Octopus server URL, API key and Space ID as output variables. This is useful when querying the Terraform state file to locate where the resources were created.")
	flags.BoolVar(&arguments.IgnoreProjectChanges, "ignoreProjectChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project (including its variables) when exporting a single project.")
	flags.BoolVar(&arguments.IgnoreProjectVariableChanges, "ignoreProjectVariableChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project's variables when exporting a single project. This differs from the ignoreProjectChanges option by only ignoring changes to variables while reapplying changes to all other project settings.")
	flags.BoolVar(&arguments.IgnoreProjectGroupChanges, "ignoreProjectGroupChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's group.")
	flags.BoolVar(&arguments.IgnoreProjectNameChanges, "ignoreProjectNameChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's name.")
	flags.BoolVar(&arguments.LookUpDefaultWorkerPools, "lookUpDefaultWorkerPools", false, "Reference the worker pool by name when a step uses the default worker pool. This means exported projects do not inherit the default worker pool when they are applied in a new space.")

	err := flags.Parse(args)

	if err != nil {
		return Arguments{}, buf.String(), err
	}

	err = overrideArgs(flags, arguments.ConfigPath, arguments.ConfigFile)

	if err != nil {
		return Arguments{}, buf.String(), err
	}

	if arguments.Url == "" {
		arguments.Url = os.Getenv("OCTOPUS_CLI_SERVER")
	}

	if arguments.ApiKey == "" {
		arguments.ApiKey = os.Getenv("OCTOPUS_CLI_API_KEY")
	}

	if err := arguments.ValidateExcludeExceptArgs(); err != nil {
		return Arguments{}, "", err
	}

	return arguments, buf.String(), nil
}

// ValidateExcludeExceptArgs removes any resource named in a Exclude<ResourceType>Except argument that does not
// exist in the Octopus instance. This is mostly used when external systems attempt to filter the results but
// may place incorrect values into the Exclude<ResourceType>Except arguments.
func (arguments *Arguments) ValidateExcludeExceptArgs() (funcErr error) {
	if !arguments.IgnoreInvalidExcludeExcept {
		return
	}

	octopusClient := client.OctopusApiClient{
		Url:    arguments.Url,
		Space:  arguments.Space,
		ApiKey: arguments.ApiKey,
	}

	filteredProjects, err := filterNamedResource[octopus.Project](octopusClient, "Projects", arguments.ExcludeProjectsExcept)

	if err != nil {
		return err
	}

	arguments.ExcludeProjectsExcept = filteredProjects

	filteredEnvironments, err := filterNamedResource[octopus.Environment](octopusClient, "Environments", arguments.ExcludeEnvironmentsExcept)

	if err != nil {
		return err
	}

	arguments.ExcludeEnvironmentsExcept = filteredEnvironments

	filteredTenants, err := filterNamedResource[octopus.Tenant](octopusClient, "Tenants", arguments.ExcludeTenantsExcept)

	if err != nil {
		return nil
	}

	arguments.ExcludeTenantsExcept = filteredTenants

	filteredMachines, err := filterNamedResource[octopus.Machine](octopusClient, "Machines", arguments.ExcludeTargetsExcept)

	if err != nil {
		return err
	}

	arguments.ExcludeTargetsExcept = filteredMachines

	filteredRunbooks, err := filterNamedResource[octopus.Runbook](octopusClient, "Runbooks", arguments.ExcludeRunbooksExcept)

	if err != nil {
		return err
	}

	arguments.ExcludeRunbooksExcept = filteredRunbooks

	filteredVariableSets, err := filterNamedResource[octopus.LibraryVariableSet](octopusClient, "LibraryVariableSets", arguments.ExcludeLibraryVariableSetsExcept)

	if err != nil {
		return err
	}

	arguments.ExcludeLibraryVariableSetsExcept = filteredVariableSets

	return err
}

func filterNamedResource[K octopus.NamedResource](octopusClient client.OctopusApiClient, resourceType string, filter []string) (results []string, funcErr error) {
	filtered := lo.Filter(filter, func(resource string, index int) bool {
		collection := octopus.GeneralCollection[K]{}
		if err := octopusClient.GetAllResources(resourceType, &collection, []string{"partialName", resource}); err != nil {
			funcErr = errors.Join(funcErr, err)
		}
		return lo.ContainsBy[K](collection.Items, func(item K) bool {
			return item.GetName() == resource
		})
	})

	return filtered, funcErr
}

// Inspired by https://github.com/carolynvs/stingoftheviper
// Viper needs manual handling to implement reading settings from env vars, config files, and from the command line
func overrideArgs(flags *flag.FlagSet, configPath string, configFile string) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(configFile)

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(configPath)

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix("octolint")

	// Environment variables can't have dashes in them, so bind them to their equivalent
	// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	return bindFlags(flags, v)
}

// Bind each flag to its associated viper configuration (config file and environment variable)
func bindFlags(flags *flag.FlagSet, v *viper.Viper) (funErr error) {
	var funcError error = nil

	flags.VisitAll(func(allFlags *flag.Flag) {
		defined := false
		flags.Visit(func(definedFlag *flag.Flag) {
			if definedFlag.Name == allFlags.Name && definedFlag.Name != "configFile" && definedFlag.Name != "configPath" {
				defined = true
			}
		})

		if !defined && v.IsSet(allFlags.Name) {
			configName := strings.ReplaceAll(allFlags.Name, "-", "")

			for _, value := range v.GetStringSlice(configName) {
				err := flags.Set(allFlags.Name, value)
				funcError = errors.Join(funcError, err)
			}
		}
	})

	return funcError
}
