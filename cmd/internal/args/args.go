package args

import (
	"flag"
	"os"
)

type Arguments struct {
	Url                          string
	ApiKey                       string
	Space                        string
	Destination                  string
	Console                      bool
	ProjectId                    string
	ProjectName                  string
	LookupProjectDependencies    bool
	IgnoreCacManagedValues       bool
	BackendBlock                 string
	DetachProjectTemplates       bool
	DefaultSecretVariableValues  bool
	ProviderVersion              string
	ExcludeAllRunbooks           bool
	ExcludeRunbooks              ExcludeRunbooks
	ExcludeProvider              bool
	ExcludeLibraryVariableSets   ExcludeLibraryVariableSets
	IgnoreProjectChanges         bool
	IgnoreProjectVariableChanges bool
	IgnoreProjectGroupChanges    bool
	IgnoreProjectNameChanges     bool
	ExcludeProjectVariables      ExcludeVariables
	ExcludeProjectVariablesRegex ExcludeVariables
}

type ExcludeVariables []string

func (i *ExcludeVariables) String() string {
	return "excluded variables"
}

func (i *ExcludeVariables) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type ExcludeRunbooks []string

func (i *ExcludeRunbooks) String() string {
	return "excluded runbooks"
}

type ExcludeLibraryVariableSets []string

func (i *ExcludeRunbooks) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ExcludeLibraryVariableSets) String() string {
	return "excluded library variable sets"
}

func (i *ExcludeLibraryVariableSets) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func ParseArgs() Arguments {
	arguments := Arguments{}

	flag.StringVar(&arguments.Url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")
	flag.StringVar(&arguments.Space, "space", "", "The Octopus space name or ID")
	flag.StringVar(&arguments.ApiKey, "apiKey", "", "The Octopus api key")
	flag.StringVar(&arguments.Destination, "dest", "", "The directory to place the Terraform files in")
	flag.BoolVar(&arguments.Console, "console", false, "Dump Terraform files to the console")
	flag.StringVar(&arguments.ProjectId, "projectId", "", "Limit the export to a single project")
	flag.StringVar(&arguments.ProjectName, "projectName", "", "Limit the export to a single project")
	flag.BoolVar(&arguments.LookupProjectDependencies, "lookupProjectDependencies", false, "Use data sources to lookup the external project dependencies. Use this when the destination space has existing environments, accounts, tenants, feeds, git credentials, and library variable sets that this project should reference.")
	flag.BoolVar(&arguments.IgnoreCacManagedValues, "ignoreCacManagedValues", false, "Pass this to exclude values managed by Config-as-Code from the exported Terraform. This includes non-sensitive variables, the deployment process, connectivity settings, and other project settings. This has no effect on projects that do not have CaC enabled.")
	flag.BoolVar(&arguments.DefaultSecretVariableValues, "defaultSecretVariableValues", false, "Pass this to set the default value of secret variables to the octostache template referencing the variable.")
	flag.StringVar(&arguments.BackendBlock, "terraformBackend", "", "Specifies the backend type to be added to the exported Terraform configuration.")
	flag.StringVar(&arguments.ProviderVersion, "providerVersion", "", "Specifies the Octopus Terraform provider version.")
	flag.BoolVar(&arguments.DetachProjectTemplates, "detachProjectTemplates", false, "Detaches any step templates in the exported Terraform.")
	flag.BoolVar(&arguments.ExcludeAllRunbooks, "excludeAllRunbooks", false, "Exclude all runbooks when exporting a project. This only takes effect when exporting a single project.")
	flag.Var(&arguments.ExcludeRunbooks, "excludeRunbook", "A runbook to be excluded when exporting a single project.")
	flag.Var(&arguments.ExcludeLibraryVariableSets, "excludeLibraryVariableSet", "A library variable set to be excluded when exporting a single project.")
	flag.Var(&arguments.ExcludeProjectVariables, "excludeProjectVariable", "Exclude a project variable from being exported.")
	flag.Var(&arguments.ExcludeProjectVariablesRegex, "excludeProjectVariableRegex", "Exclude a project variable from being exported based on regex match.")
	flag.BoolVar(&arguments.ExcludeProvider, "excludeProvider", false, "Exclude the provider from the exported Terraform configuration files. This is useful when you want to use a parent module to define the backend, as the parent module must define the provider.")
	flag.BoolVar(&arguments.IgnoreProjectChanges, "ignoreProjectChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project (including its variables) when exporting a single project.")
	flag.BoolVar(&arguments.IgnoreProjectVariableChanges, "ignoreProjectVariableChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project's variables when exporting a single project. This differs from the ignoreProjectChanges option by only ignoring changes to variables while reapplying changes to all other project settings.")
	flag.BoolVar(&arguments.IgnoreProjectGroupChanges, "ignoreProjectGroupChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's group.")
	flag.BoolVar(&arguments.IgnoreProjectNameChanges, "ignoreProjectNameChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's name.")
	flag.Parse()

	if arguments.Url == "" {
		arguments.Url = os.Getenv("OCTOPUS_CLI_SERVER")
	}

	if arguments.ApiKey == "" {
		arguments.ApiKey = os.Getenv("OCTOPUS_CLI_API_KEY")
	}

	return arguments
}
