package args

import (
	"bytes"
	"flag"
	"os"
	"strings"
)

type Arguments struct {
	Version                          bool
	Url                              string
	ApiKey                           string
	Space                            string
	Destination                      string
	Console                          bool
	ProjectId                        string
	ProjectName                      string
	LookupProjectDependencies        bool
	IgnoreCacManagedValues           bool
	BackendBlock                     string
	DetachProjectTemplates           bool
	DefaultSecretVariableValues      bool
	DummySecretVariableValues        bool
	ProviderVersion                  string
	ExcludeAllRunbooks               bool
	ExcludeRunbooks                  ExcludeRunbooks
	ExcludeRunbooksRegex             ExcludeRunbooks
	ExcludeProvider                  bool
	IncludeOctopusOutputVars         bool
	ExcludeLibraryVariableSets       ExcludeLibraryVariableSets
	ExcludeLibraryVariableSetsRegex  ExcludeLibraryVariableSets
	IgnoreProjectChanges             bool
	IgnoreProjectVariableChanges     bool
	IgnoreProjectGroupChanges        bool
	IgnoreProjectNameChanges         bool
	ExcludeProjectVariables          ExcludeVariables
	ExcludeProjectVariablesRegex     ExcludeVariables
	ExcludeVariableEnvironmentScopes ExcludeVariableEnvironmentScopes
	LookUpDefaultWorkerPools         bool
	ExcludeTenantTags                ExcludeTenantTags
	ExcludeTenants                   ExcludeTenants
	ExcludeTenantsWithTags           ExcludeTenantsWithTags
	ExcludeTenantsExcept             ExcludeTenantsExcept
	ExcludeAllTenants                bool
	ExcludeProjects                  ExcludeProjects
	ExcludeProjectsRegex             ExcludeProjectsRegex
	ExcludeAllProjects               bool
	ExcludeAllTargets                bool
}

type ExcludeProjects []string

func (i *ExcludeProjects) String() string {
	return "excluded projects"
}

func (i *ExcludeProjects) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeProjectsRegex []string

func (i *ExcludeProjectsRegex) String() string {
	return "excluded projects"
}

func (i *ExcludeProjectsRegex) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeTenantsExcept []string

func (i *ExcludeTenantsExcept) String() string {
	return "exclude tenants except"
}

func (i *ExcludeTenantsExcept) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeTenants []string

func (i *ExcludeTenants) String() string {
	return "excluded tenants"
}

func (i *ExcludeTenants) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeTenantTags []string

func (i *ExcludeTenantTags) String() string {
	return "excluded tenant tags"
}

func (i *ExcludeTenantTags) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeTenantsWithTags []string

func (i *ExcludeTenantsWithTags) String() string {
	return "excluded tenantwith tag"
}

func (i *ExcludeTenantsWithTags) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeVariableEnvironmentScopes []string

func (i *ExcludeVariableEnvironmentScopes) String() string {
	return "excluded variable environment scopes"
}

func (i *ExcludeVariableEnvironmentScopes) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeVariables []string

func (i *ExcludeVariables) String() string {
	return "excluded variables"
}

func (i *ExcludeVariables) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

type ExcludeRunbooks []string

func (i *ExcludeRunbooks) String() string {
	return "excluded runbooks"
}

type ExcludeLibraryVariableSets []string

func (i *ExcludeRunbooks) Set(value string) error {
	trimmed := strings.TrimSpace(value)

	if len(trimmed) == 0 {
		return nil
	}

	*i = append(*i, trimmed)
	return nil
}

func (i *ExcludeLibraryVariableSets) String() string {
	return "excluded library variable sets"
}

func (i *ExcludeLibraryVariableSets) Set(value string) error {
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

	flags.BoolVar(&arguments.Version, "version", false, "Print the version")
	flags.StringVar(&arguments.Url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")
	flags.StringVar(&arguments.Space, "space", "", "The Octopus space name or ID")
	flags.StringVar(&arguments.ApiKey, "apiKey", "", "The Octopus api key")
	flags.StringVar(&arguments.Destination, "dest", "", "The directory to place the Terraform files in")
	flags.BoolVar(&arguments.Console, "console", false, "Dump Terraform files to the console")
	flags.StringVar(&arguments.ProjectId, "projectId", "", "Limit the export to a single project")
	flags.StringVar(&arguments.ProjectName, "projectName", "", "Limit the export to a single project")
	flags.BoolVar(&arguments.LookupProjectDependencies, "lookupProjectDependencies", false, "Use data sources to lookup the external project dependencies. Use this when the destination space has existing environments, accounts, tenants, feeds, git credentials, and library variable sets that this project should reference.")
	flags.BoolVar(&arguments.IgnoreCacManagedValues, "ignoreCacManagedValues", true, "Pass this to exclude values managed by Config-as-Code from the exported Terraform. This includes non-sensitive variables, the deployment process, connectivity settings, and other project settings. This has no effect on projects that do not have CaC enabled.")
	flags.BoolVar(&arguments.DefaultSecretVariableValues, "defaultSecretVariableValues", false, "Pass this to set the default value of secret variables to the octostache template referencing the variable.")
	flags.BoolVar(&arguments.DummySecretVariableValues, "dummySecretVariableValues", false, "Pass this to set the default value of secret variables, account secrets, feed credentials to a dummy value. This allows resources with secret values to be created without knowing the secrets, while still allowing the secret values to be specified if they are known.")
	flags.StringVar(&arguments.BackendBlock, "terraformBackend", "", "Specifies the backend type to be added to the exported Terraform configuration.")
	flags.StringVar(&arguments.ProviderVersion, "providerVersion", "", "Specifies the Octopus Terraform provider version.")
	flags.BoolVar(&arguments.DetachProjectTemplates, "detachProjectTemplates", false, "Detaches any step templates in the exported Terraform.")
	flags.BoolVar(&arguments.ExcludeAllRunbooks, "excludeAllRunbooks", false, "Exclude all runbooks when exporting a project. This only takes effect when exporting a single project.")
	flags.Var(&arguments.ExcludeRunbooks, "excludeRunbook", "A runbook to be excluded when exporting a single project.")
	flags.Var(&arguments.ExcludeRunbooksRegex, "excludeRunbookRegex", "A runbook to be excluded when exporting a single project based on regex match.")
	flags.Var(&arguments.ExcludeLibraryVariableSets, "excludeLibraryVariableSet", "A library variable set to be excluded when exporting a single project.")
	flags.Var(&arguments.ExcludeLibraryVariableSetsRegex, "excludeLibraryVariableSetRegex", "A library variable set to be excluded when exporting a single project based on regex match.")
	flags.Var(&arguments.ExcludeProjectVariables, "excludeProjectVariable", "Exclude a project variable from being exported.")
	flags.Var(&arguments.ExcludeProjectVariablesRegex, "excludeProjectVariableRegex", "Exclude a project variable from being exported based on regex match.")
	flags.Var(&arguments.ExcludeVariableEnvironmentScopes, "excludeVariableEnvironmentScopes", "Exclude a environment when it appears in a variable's environment scope. Use with caution, as this can lead to previously scoped variables becoming unscoped.")
	flags.Var(&arguments.ExcludeTenantTags, "excludeTenantTags", "Exclude a tenant tag from being exported. Tags are in the format \"taggroup/tagname\".")
	flags.Var(&arguments.ExcludeTenants, "excludeTenants", "Exclude a tenant from being exported.")
	flags.Var(&arguments.ExcludeTenantsWithTags, "excludeTenantsWithTag", "Exclude any tenant with this tag from being exported. This is useful when using tags to separate tenants that can be exported with those that should not.")
	flags.Var(&arguments.ExcludeTenantsExcept, "excludeTenantsExcept", "Exclude all tenants except for those define in this list. The tenants in excludeTenants take precedence, so a tenant define here and in excludeTenants is excluded.")
	flags.Var(&arguments.ExcludeProjects, "excludeProjects", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.Var(&arguments.ExcludeProjectsRegex, "excludeProjectsRegex", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.BoolVar(&arguments.ExcludeAllProjects, "excludeAllProjects", false, "Exclude all projects from being exported. This is only used when exporting a space.")
	flags.BoolVar(&arguments.ExcludeProvider, "excludeProvider", false, "Exclude the provider from the exported Terraform configuration files. This is useful when you want to use a parent module to define the backend, as the parent module must define the provider.")
	flags.BoolVar(&arguments.IncludeOctopusOutputVars, "includeOctopusOutputVars", true, "Capture the Octopus server URL, API key and Space ID as output variables. This is useful when querying the Terraform state file to locate where the resources were created.")
	flags.BoolVar(&arguments.IgnoreProjectChanges, "ignoreProjectChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project (including its variables) when exporting a single project.")
	flags.BoolVar(&arguments.IgnoreProjectVariableChanges, "ignoreProjectVariableChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project's variables when exporting a single project. This differs from the ignoreProjectChanges option by only ignoring changes to variables while reapplying changes to all other project settings.")
	flags.BoolVar(&arguments.IgnoreProjectGroupChanges, "ignoreProjectGroupChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's group.")
	flags.BoolVar(&arguments.IgnoreProjectNameChanges, "ignoreProjectNameChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's name.")
	flags.BoolVar(&arguments.LookUpDefaultWorkerPools, "lookUpDefaultWorkerPools", false, "Reference the worker pool by name when a step uses the default worker pool. This means exported projects do not inherit the default worker pool when they are applied in a new space.")
	flags.BoolVar(&arguments.ExcludeAllTenants, "excludeAllTenants", false, "Exclude all tenants from being exported.")
	flags.BoolVar(&arguments.ExcludeAllTargets, "excludeAllTargets", false, "Exclude all targets from being exported.")
	err := flags.Parse(args)

	if err != nil {
		return Arguments{}, buf.String(), err
	}

	if arguments.Url == "" {
		arguments.Url = os.Getenv("OCTOPUS_CLI_SERVER")
	}

	if arguments.ApiKey == "" {
		arguments.ApiKey = os.Getenv("OCTOPUS_CLI_API_KEY")
	}

	return arguments, buf.String(), nil
}
