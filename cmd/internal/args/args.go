package args

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/types"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type Arguments struct {
	InsecureTls                     *bool            `json:"insecureTls,omitempty" jsonschema:"Ignore certificate errors when connecting to the Octopus server."`
	ExperimentalEnableStepTemplates *bool            `json:"experimentalEnableStepTemplates,omitempty" jsonschema:"Has no effect. This option used to enable the export of step templates, but this is now a standard feature. This option is left in for compatibility."`
	Profiling                       *bool            `json:"profiling,omitempty" jsonschema:"Enable profiling. Run 'pprof -http=:8080 octoterra.prof' to view the results."`
	ExcludeTerraformVariables       *bool            `json:"excludeTerraformVariables,omitempty" jsonschema:"This option means the exported module does not expose Terraform variables for common inputs like the value of project or library variables set variables. This reduces the size of the Terraform configuration files, but makes the module less configurable because values are hard coded."`
	ExcludeSpaceCreation            *bool            `json:"excludeSpaceCreation,omitempty" jsonschema:"This option excludes the Terraform configuration that is used to create the space."`
	ConfigFile                      *string          `json:"configFile,omitempty" jsonschema:"The name of the configuration file to use. Do not include the extension. Defaults to octoterra"`
	ConfigPath                      *string          `json:"configPath,omitempty" jsonschema:"The path of the configuration file to use. Defaults to the current directory"`
	Version                         *bool            `json:"version,omitempty" jsonschema:"Print the version"`
	IgnoreInvalidExcludeExcept      *bool            `json:"ignoreInvalidExcludeExcept,omitempty" jsonschema:"Ensures that resource names passed to the 'Exclude<ResourceType>Except' arguments are valid, and if they are not, removes those names from the list. This is useful when an external system attempts to filter results but places incorrect values into 'Exclude<ResourceType>Except' arguments. It may result in all resources being returned if no valid resources names are included in the 'Exclude<ResourceType>Except' arguments."`
	Url                             *string          `json:"url,omitempty" jsonschema:"The Octopus URL e.g. https://myinstance.octopus.app - this is also defined in the OCTOPUS_CLI_SERVER environment variable. Do not set this when calling as an MCP tool."`
	ApiKey                          *string          `json:"apiKey,omitempty" jsonschema:"The Octopus api key - this is also defined in the OCTOPUS_CLI_API_KEY environment variable. Do not set this when calling as an MCP tool."`
	AccessToken                     *string          `json:"accessToken,omitempty" jsonschema:"The Octopus access token"`
	UseRedirector                   *bool            `json:"useRedirector,omitempty" jsonschema:"Set to true to access the Octopus instance via the redirector"`
	RedirectorHost                  *string          `json:"redirectorHost,omitempty" jsonschema:"The hostname of the redirector service"`
	RedirectorServiceApiKey         *string          `json:"redirectorServiceApiKey,omitempty" jsonschema:"The service api key of the redirector service"`
	RedirecrtorApiKey               *string          `json:"redirecrtorApiKey,omitempty" jsonschema:"The user api key of the redirector service"`
	RedirectorRedirections          *string          `json:"redirectorRedirections,omitempty" jsonschema:"The redirection rules for the redirector service"`
	Space                           *string          `json:"space,omitempty" jsonschema:"The Octopus space name or ID"`
	Destination                     *string          `json:"dest,omitempty" jsonschema:"The directory to place the Terraform files in"`
	Console                         *bool            `json:"console,omitempty" jsonschema:"Dump Terraform files to the console"`
	ProjectId                       *StringSliceArgs `json:"projectId,omitempty" jsonschema:"Limit the export to a single project"`
	ProjectName                     *StringSliceArgs `json:"projectName,omitempty" jsonschema:"Limit the export to a single project"`
	RunbookId                       *string          `json:"runbookId,omitempty" jsonschema:"Limit the export to a single runbook. Runbooks are exported referencing external resources as data sources."`
	RunbookName                     *string          `json:"runbookName,omitempty" jsonschema:"Limit the export to a single runbook. Requires projectName or projectId. Runbooks are exported referencing external resources as data sources."`
	LookupProjectDependencies       *bool            `json:"lookupProjectDependencies,omitempty" jsonschema:"Use data sources to lookup the external project dependencies. Use this when the destination space has existing environments, accounts, tenants, feeds, git credentials, and library variable sets that this project should reference."`
	LookupProjectLinkTenants        *bool            `json:"lookupProjectLinkTenants,omitempty" jsonschema:"When lookupProjectDependencies is true, lookupProjectLinkTenants will reestablish the link to tenants that were linked to the source project and recreate any project and common tenant variables. Essentially this means the exported project 'owns' the relationship to the tenant and any variables used by the tenant."`
	Stateless                       *bool            `json:"stepTemplate,omitempty" jsonschema:"Create an Octopus step template"`
	StatelessAdditionalParams       *StringSliceArgs `json:"stepTemplateAdditionalParameters,omitempty" jsonschema:"Indicates that a non-secret variable should be exposed as a parameter. The format of this option is 'ProjectName:VariableName'. This option is only used with the -stepTemplate option."`
	StepTemplateName                *string          `json:"stepTemplateName,omitempty" jsonschema:"Step template name. Only used with the stepTemplate option."`
	StepTemplateKey                 *string          `json:"stepTemplateKey,omitempty" jsonschema:"Step template key used when building parameter names. Only used with the stepTemplate option."`
	StepTemplateDescription         *string          `json:"stepTemplateDescription,omitempty" jsonschema:"Step template description used when building parameter names. Only used with the stepTemplate option."`
	IgnoreCacManagedValues          *bool            `json:"ignoreCacManagedValues,omitempty" jsonschema:"Pass this to exclude values managed by Config-as-Code from the exported Terraform. This includes non-sensitive variables, the deployment process, connectivity settings, and other project settings. This has no effect on projects that do not have CaC enabled."`
	ExcludeCaCProjectSettings       *bool            `json:"excludeCaCProjectSettings,omitempty" jsonschema:"Pass this to exclude any Config-As-Code settings in the exported projects. Typically you set -ignoreCacManagedValues=false -excludeCaCProjectSettings=true to essentially 'convert' a CaC project to a regular project. Values from the 'main' or 'master' branches will be used first, or just fall back to the first configured branch."`
	BackendBlock                    *string          `json:"terraformBackend,omitempty" jsonschema:"Specifies the backend type to be added to the exported Terraform configuration."`
	DetachProjectTemplates          *bool            `json:"detachProjectTemplates,omitempty" jsonschema:"Detaches any step templates in the exported Terraform."`
	DefaultSecretVariableValues     *bool            `json:"defaultSecretVariableValues,omitempty" jsonschema:"Pass this to set the default value of secret variables to the octostache template referencing the variable."`
	DummySecretVariableValues       *bool            `json:"dummySecretVariableValues,omitempty" jsonschema:"Pass this to set the default value of secret variables, account secrets, feed credentials to a dummy value. This allows resources with secret values to be created without knowing the secrets, while still allowing the secret values to be specified if they are known. This option takes precedence over the defaultSecretVariableValues option."`
	InlineVariableValues            *bool            `json:"inlineVariableValues,omitempty" jsonschema:"Inline the project and library variable set variable values rather than exposing their value as a Terraform variable. Secret variables will be inlined as dummy values. This option takes precedence over DummySecretVariableValues and DefaultSecretVariableValues."`
	ProviderVersion                 *string          `json:"providerVersion,omitempty" jsonschema:"Specifies the Octopus Terraform provider version."`
	ExcludeProvider                 *bool            `json:"excludeProvider,omitempty" jsonschema:"Exclude the provider from the exported Terraform configuration files. This is useful when you want to use a parent module to define the backend, as the parent module must define the provider."`
	IncludeProviderServerDetails    *bool            `json:"includeProviderServerDetails,omitempty" jsonschema:"Define the server URL and API keys as variables passed to the provider. Set this to false to use the OCTOPUS_ACCESS_TOKEN, OCTOPUS_URL, and OCTOPUS_APIKEY environment variables to configure the provider."`
	IncludeOctopusOutputVars        *bool            `json:"includeOctopusOutputVars,omitempty" jsonschema:"Capture the Octopus server URL, API key and Space ID as output variables. This is useful when querying the Terraform state file to locate where the resources were created."`
	LimitAttributeLength            *int             `json:"limitAttributeLength,omitempty" jsonschema:"For internal use only. Limits the length of the attribute names."`
	LimitResourceCount              *int             `json:"limitResourceCount,omitempty" jsonschema:"For internal use only. Limits the number of resources of a given type that are returned. For example, a value of 30 will ensure the exported Terraform only includes up to 30 accounts, and up to 30 feeds, and up to 30 projects etc. This is used to reduce the output when octoterra is used to generate a context for an LLM. This limit is a guide and it is possible that more than the specified number of resources are returned due to multiple goroutines adding resources to the output."`
	GenerateImportScripts           *bool            `json:"generateImportScripts,omitempty" jsonschema:"Generate Bash and Powershell scripts used to import resources into the Terraform state."`
	IgnoreCacErrors                 *bool            `json:"ignoreCacErrors,omitempty" jsonschema:"Ignores errors that would arise when a project can not resolve configuration in a Git repo."`

	OctopusManagedTerraformVars *string `json:"octopusManagedTerraformVars,omitempty" jsonschema:"Specifies the name of an Octopus variable to be used as a template string in the body of the terraform.tfvars file. This allows Octopus to inject all the variables used by Terraform from a variable containing the contents of a terraform.tfvars file."`

	IgnoreProjectChanges         *bool `json:"ignoreProjectChanges,omitempty" jsonschema:"Use the Terraform lifecycle meta-argument to ignore all changes to the project (including its variables) when exporting a single project."`
	IgnoreProjectVariableChanges *bool `json:"ignoreProjectVariableChanges,omitempty" jsonschema:"Use the Terraform lifecycle meta-argument to ignore all changes to the project's variables when exporting a single project. This differs from the ignoreProjectChanges option by only ignoring changes to variables while reapplying changes to all other project settings."`
	IgnoreProjectGroupChanges    *bool `json:"ignoreProjectGroupChanges,omitempty" jsonschema:"Use the Terraform lifecycle meta-argument to ignore the changes to the project's group."`
	IgnoreProjectNameChanges     *bool `json:"ignoreProjectNameChanges,omitempty" jsonschema:"Use the Terraform lifecycle meta-argument to ignore the changes to the project's name."`
	LookUpDefaultWorkerPools     *bool `json:"lookUpDefaultWorkerPools,omitempty" jsonschema:"Reference the worker pool by name when a step uses the default worker pool. This means exported projects do not inherit the default worker pool when they are applied in a new space."`
	IncludeIds                   *bool `json:"includeIds,omitempty" jsonschema:"For internal use only. Include the 'id' field on generated resources. Note that this is almost always unnecessary and undesirable."`
	IncludeSpaceInPopulation     *bool `json:"includeSpaceInPopulation,omitempty" jsonschema:"For internal use only. Include the space resource in the space population script. Note that this is almost always unnecessary and undesirable, as the space resources are included in the space creation module."`
	IncludeDefaultChannel        *bool `json:"includeDefaultChannel,omitempty" jsonschema:"Internal use only. Includes the 'Default' channel as a standard channel resource rather than a data block."`

	/*
		We expose a lot of options to exclude resources from the export.

		Some of these exclusions are "natural" in that you can simply not export them and the resulting Terraform configuration will be (mostly) valid.
		For example, you can exclude targets and runbooks and the only impact is that some scoped variables may become unscoped. Or you can exclude
		projects and only "Deploy a Release" steps will be impacted.

		Other exclusions are "core" in that they are tightly coupled to many other resources. For example, excluding environments, lifecycles,
		project groups, accounts etc. will likely leave the resulting Terraform configuration in an invalid state.

		Warnings have been added to the CLI help command to identify some of the side-effects of the various exclusions.
	*/

	ExcludeAllProjectVariables       *bool            `json:"excludeAllProjectVariables,omitempty" jsonschema:"Exclude all project variables from being exported. WARNING: steps that used this variable may no longer function correctly."`
	ExcludeProjectVariables          *StringSliceArgs `json:"excludeProjectVariable,omitempty" jsonschema:"Exclude a project variable from being exported. WARNING: steps that used this variable may no longer function correctly."`
	ExcludeProjectVariablesExcept    *StringSliceArgs `json:"excludeProjectVariableExcept,omitempty" jsonschema:"All project variables except those defined with excludeProjectVariableExcept are excluded. WARNING: steps that used other variables may no longer function correctly."`
	ExcludeProjectVariablesRegex     *StringSliceArgs `json:"excludeProjectVariableRegex,omitempty" jsonschema:"Exclude a project variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly."`
	ExcludeVariableEnvironmentScopes *StringSliceArgs `json:"excludeVariableEnvironmentScopes,omitempty" jsonschema:"Exclude an environment when it appears in a variable's environment scope. WARNING: variables scoped to this environment will no longer have that environment scope applied."`

	ExcludeAllTenantVariables    *bool            `json:"excludeAllTenantVariables,omitempty" jsonschema:"Exclude all tenant variables from being exported. WARNING: steps that used this variable may no longer function correctly."`
	ExcludeTenantVariables       *StringSliceArgs `json:"excludeTenantVariables,omitempty" jsonschema:"Exclude a tenant variable from being exported. WARNING: steps that used this variable may no longer function correctly."`
	ExcludeTenantVariablesExcept *StringSliceArgs `json:"excludeTenantVariablesExcept,omitempty" jsonschema:"All tenant variables except those defined with excludeTenantVariablesExcept are excluded. WARNING: steps that used other variables may no longer function correctly."`
	ExcludeTenantVariablesRegex  *StringSliceArgs `json:"excludeTenantVariablesRegex,omitempty" jsonschema:"Exclude a tenant variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly."`

	ExcludeAllSteps    *bool            `json:"excludeAllSteps,omitempty" jsonschema:"Exclude all steps when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied."`
	ExcludeSteps       *StringSliceArgs `json:"excludeSteps,omitempty" jsonschema:"A step to be excluded when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied."`
	ExcludeStepsRegex  *StringSliceArgs `json:"excludeStepsRegex,omitempty" jsonschema:"A step to be excluded when exporting projects or runbooks based on regex match. WARNING: variables scoped to this step will no longer have the step scope applied."`
	ExcludeStepsExcept *StringSliceArgs `json:"excludeStepsExcept,omitempty" jsonschema:"All steps except those defined with excludeStepsExcept are excluded when exporting a project or runbook. WARNING: variables scoped to other steps will no longer have the step scope applied."`

	ExcludeAllChannels     *bool            `json:"excludeAllChannels,omitempty" jsonschema:"Exclude all channels from being exported. WARNING: Variables and steps that were scoped to channels will become unscoped."`
	ExcludeChannels        *StringSliceArgs `json:"excludeChannels,omitempty" jsonschema:"Exclude a channel from being exported. WARNING: Variables and steps that were scoped to channels will become unscoped."`
	ExcludeChannelsRegex   *StringSliceArgs `json:"excludeChannelsRegex,omitempty" jsonschema:"Exclude a channel from being exported based on regex match. WARNING: Variables and steps that were scoped to channels will become unscoped."`
	ExcludeChannelsExcept  *StringSliceArgs `json:"excludeChannelsExcept,omitempty" jsonschema:"All channels except those defined with excludeChannelsExcept are excluded. WARNING: Variables and steps that were scoped to channels will become unscoped."`
	ExcludeInvalidChannels *bool            `json:"excludeInvalidChannels,omitempty" jsonschema:"Channels that reference packages that are no longer defined in the deployment process will be excluded. WARNING: Variables and steps that were scoped to channels will become unscoped."`

	ExcludeAllRunbooks    *bool            `json:"excludeAllRunbooks,omitempty" jsonschema:"Exclude all runbooks when exporting a project or space. WARNING: variables scoped to this runbook will no longer have the runbook scope applied."`
	ExcludeRunbooks       *StringSliceArgs `json:"excludeRunbook,omitempty" jsonschema:"A runbook to be excluded when exporting a single project. WARNING: variables scoped to this runbook will no longer have the runbook scope applied."`
	ExcludeRunbooksRegex  *StringSliceArgs `json:"excludeRunbookRegex,omitempty" jsonschema:"A runbook to be excluded when exporting a single project based on regex match. WARNING: variables scoped to this runbook will no longer have the runbook scope applied."`
	ExcludeRunbooksExcept *StringSliceArgs `json:"excludeRunbooksExcept,omitempty" jsonschema:"All runbooks except those defined with excludeRunbooksExcept are excluded when exporting a single project. WARNING: variables scoped to other runbooks will no longer have the runbook scope applied."`

	ExcludeAllTriggers    *bool            `json:"excludeAllTriggers,omitempty" jsonschema:"Exclude all triggers when exporting a project or space."`
	ExcludeTriggers       *StringSliceArgs `json:"excludeTrigger,omitempty" jsonschema:"A trigger to be excluded when exporting a single project."`
	ExcludeTriggersRegex  *StringSliceArgs `json:"excludeTriggerRegex,omitempty" jsonschema:"A trigger to be excluded when exporting a single project based on regex match."`
	ExcludeTriggersExcept *StringSliceArgs `json:"excludeTriggersExcept,omitempty" jsonschema:"All triggers except those defined with excludeTriggersExcept are excluded when exporting a single project."`

	ExcludeLibraryVariableSets       *StringSliceArgs `json:"excludeLibraryVariableSet,omitempty" jsonschema:"A library variable set to be excluded when exporting a single project. WARNING: projects that linked this library variable set will no longer include these variables."`
	ExcludeLibraryVariableSetsRegex  *StringSliceArgs `json:"excludeLibraryVariableSetRegex,omitempty" jsonschema:"A library variable set to be excluded when exporting a single project based on regex match. WARNING: projects that linked this library variable set will no longer include these variables."`
	ExcludeLibraryVariableSetsExcept *StringSliceArgs `json:"excludeLibraryVariableSetsExcept,omitempty" jsonschema:"All library variable sets except those defined with excludeLibraryVariableSetsExcept are excluded. WARNING: projects that linked other library variable sets will no longer include these variables."`
	ExcludeAllLibraryVariableSets    *bool            `json:"excludeAllLibraryVariableSets,omitempty" jsonschema:"Exclude all library variable sets. WARNING: projects that linked this library variable set will no longer include these variables."`

	ExcludeEnvironments       *StringSliceArgs `json:"excludeEnvironments,omitempty" jsonschema:"An environment to be excluded when exporting a single project. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeEnvironmentsRegex  *StringSliceArgs `json:"excludeEnvironmentsRegex,omitempty" jsonschema:"An environment to be excluded when exporting a single project based on regex match. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeEnvironmentsExcept *StringSliceArgs `json:"excludeEnvironmentsExcept,omitempty" jsonschema:"All environments except those defined with excludeEnvironmentsExcept are excluded. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllEnvironments    *bool            `json:"excludeAllEnvironments,omitempty" jsonschema:"Exclude all environments. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeFeeds       *StringSliceArgs `json:"excludeFeeds,omitempty" jsonschema:"A feed to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeFeedsRegex  *StringSliceArgs `json:"excludeFeedsRegex,omitempty" jsonschema:"A feed to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeFeedsExcept *StringSliceArgs `json:"excludeFeedsExcept,omitempty" jsonschema:"All feeds except those defined with excludeFeedsExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllFeeds    *bool            `json:"excludeAllFeeds,omitempty" jsonschema:"Exclude all feeds. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeProjectGroups       *StringSliceArgs `json:"excludeProjectGroups,omitempty" jsonschema:"A project group to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeProjectGroupsRegex  *StringSliceArgs `json:"excludeProjectGroupsRegex,omitempty" jsonschema:"A project group to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeProjectGroupsExcept *StringSliceArgs `json:"excludeProjectGroupsExcept,omitempty" jsonschema:"All project groups except those defined with excludeProjectGroupsExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllProjectGroups    *bool            `json:"excludeAllProjectGroups,omitempty" jsonschema:"Exclude all project groups. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeAccounts       *StringSliceArgs `json:"excludeAccounts,omitempty" jsonschema:"An account to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAccountsRegex  *StringSliceArgs `json:"excludeAccountsRegex,omitempty" jsonschema:"An account to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAccountsExcept *StringSliceArgs `json:"excludeAccountsExcept,omitempty" jsonschema:"All accounts except those defined with excludeAccountsExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllAccounts    *bool            `json:"excludeAllAccounts,omitempty" jsonschema:"Exclude all accounts. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeCertificates       *StringSliceArgs `json:"excludeCertificates,omitempty" jsonschema:"A certificate to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeCertificatesRegex  *StringSliceArgs `json:"excludeCertificatesRegex,omitempty" jsonschema:"A certificate to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeCertificatesExcept *StringSliceArgs `json:"excludeCertificatesExcept,omitempty" jsonschema:"All certificates except those defined with excludeCertificatesExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllCertificates    *bool            `json:"excludeAllCertificates,omitempty" jsonschema:"Exclude all certificates. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeLifecycles       *StringSliceArgs `json:"excludeLifecycles,omitempty" jsonschema:"A lifecycle to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeLifecyclesRegex  *StringSliceArgs `json:"excludeLifecyclesRegex,omitempty" jsonschema:"A lifecycle to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeLifecyclesExcept *StringSliceArgs `json:"excludeLifecyclesExcept,omitempty" jsonschema:"All lifecycles except those defined with excludeLifecyclesExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllLifecycles    *bool            `json:"excludeAllLifecycles,omitempty" jsonschema:"Exclude all lifecycles. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeWorkerpools       *StringSliceArgs `json:"excludeWorkerPools,omitempty" jsonschema:"A worker pool to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeWorkerpoolsRegex  *StringSliceArgs `json:"excludeWorkerPoolsRegex,omitempty" jsonschema:"A worker pool to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeWorkerpoolsExcept *StringSliceArgs `json:"excludeWorkerPoolsExcept,omitempty" jsonschema:"All worker pools except those defined with excludeWorkerPoolsExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllWorkerpools    *bool            `json:"excludeAllWorkerPools,omitempty" jsonschema:"Exclude all worker pools. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeMachinePolicies       *StringSliceArgs `json:"excludeMachinePolicies,omitempty" jsonschema:"A machine policy to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeMachinePoliciesRegex  *StringSliceArgs `json:"excludeMachinePoliciesRegex,omitempty" jsonschema:"A machine policy to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeMachinePoliciesExcept *StringSliceArgs `json:"excludeMachinePoliciesExcept,omitempty" jsonschema:"All machine policies except those defined with excludeMachinePoliciesExcept are excluded. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`
	ExcludeAllMachinePolicies    *bool            `json:"excludeAllMachinePolicies,omitempty" jsonschema:"Exclude all machine policies. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled."`

	ExcludeMachineProxies       *StringSliceArgs `json:"excludeMachineProxies,omitempty" jsonschema:"A machine proxy to be excluded when exporting a single project."`
	ExcludeMachineProxiesRegex  *StringSliceArgs `json:"excludeMachineProxiesRegex,omitempty" jsonschema:"A machine proxy to be excluded when exporting a single project based on regex match."`
	ExcludeMachineProxiesExcept *StringSliceArgs `json:"excludeMachineProxiesExcept,omitempty" jsonschema:"All machine proxies except those defined with excludeMachineProxiesExcept are excluded."`
	ExcludeAllMachineProxies    *bool            `json:"excludeAllMachineProxies,omitempty" jsonschema:"Exclude all machine proxies."`

	ExcludeTenantTags *StringSliceArgs `json:"excludeTenantTags,omitempty" jsonschema:"Exclude an individual tenant tag from being exported. Tags are in the format 'taggroup/tagname'. WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied."`

	ExcludeTenantTagSets       *StringSliceArgs `json:"excludeTenantTagSets,omitempty" jsonschema:"Exclude a tenant tag set from being exported. WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied."`
	ExcludeTenantTagSetsRegex  *StringSliceArgs `json:"excludeTenantTagSetsRegex,omitempty" jsonschema:"Exclude tenant tag sets from being exported based on a regex. WARNING: Resources scoped to tenant tag sets will be unscoped."`
	ExcludeTenantTagSetsExcept *StringSliceArgs `json:"excludeTenantTagSetsExcept,omitempty" jsonschema:"Exclude all tenant tag sets except for those defined in this list. WARNING: Resources scoped to tenant tag sets will be unscoped."`
	ExcludeAllTenantTagSets    *bool            `json:"excludeAllTenantTagSets,omitempty" jsonschema:"Exclude all tenant tag sets from being exported. WARNING: Resources scoped to tenant tag sets will be unscoped."`

	ExcludeTenants         *StringSliceArgs `json:"excludeTenants,omitempty" jsonschema:"Exclude a tenant from being exported."`
	ExcludeTenantsRegex    *StringSliceArgs `json:"excludeTenantsRegex,omitempty" jsonschema:"Exclude a tenant from being exported based on a regex."`
	ExcludeTenantsWithTags *StringSliceArgs `json:"excludeTenantsWithTag,omitempty" jsonschema:"Exclude any tenant with this tag from being exported. This is useful when using tags to separate tenants that can be exported with those that should not. Tags are in the format TagGroupName/TagName."`
	ExcludeTenantsExcept   *StringSliceArgs `json:"excludeTenantsExcept,omitempty" jsonschema:"Exclude all tenants except for those defined in this list. The tenants in excludeTenants take precedence, so a tenant defined here and in excludeTenants is excluded."`
	ExcludeAllTenants      *bool            `json:"excludeAllTenants,omitempty" jsonschema:"Exclude all tenants from being exported."`

	ExcludeProjects       *StringSliceArgs `json:"excludeProjects,omitempty" jsonschema:"Exclude a project from being exported. This is only used when exporting a space."`
	ExcludeProjectsExcept *StringSliceArgs `json:"excludeProjectsExcept,omitempty" jsonschema:"All projects except those defined with excludeProjectsExcept are excluded. This is only used when exporting a space."`
	ExcludeProjectsRegex  *StringSliceArgs `json:"excludeProjectsRegex,omitempty" jsonschema:"Exclude a project from being exported based on regex match. This is only used when exporting a space."`
	ExcludeAllProjects    *bool            `json:"excludeAllProjects,omitempty" jsonschema:"Exclude all projects from being exported. This is only used when exporting a space."`

	ExcludeAllTargets                *bool            `json:"excludeAllTargets,omitempty" jsonschema:"Exclude all targets from being exported. WARNING: Variables that were scoped to targets will become unscoped."`
	ExcludeTargets                   *StringSliceArgs `json:"excludeTargets,omitempty" jsonschema:"Exclude targets from being exported. WARNING: Variables that were scoped to targets will become unscoped."`
	ExcludeTargetsRegex              *StringSliceArgs `json:"excludeTargetsRegex,omitempty" jsonschema:"Exclude targets from being exported based on a regex. WARNING: Variables that were scoped to targets will become unscoped."`
	ExcludeTargetsExcept             *StringSliceArgs `json:"excludeTargetsExcept,omitempty" jsonschema:"Exclude all targets except for those defined in this list. The targets in excludeTargets take precedence, so a target defined here and in excludeTargets is excluded. WARNING: Variables that were scoped to other targets will become unscoped."`
	ExcludeTargetsWithNoEnvironments *bool            `json:"excludeTargetsWithNoEnvironments,omitempty" jsonschema:"Exclude targets that have had all their environments excluded. WARNING: Variables that were scoped to targets will become unscoped."`

	ExcludeAllWorkers    *bool            `json:"excludeAllWorkers,omitempty" jsonschema:"Exclude all workers from being exported."`
	ExcludeWorkers       *StringSliceArgs `json:"excludeWorkers,omitempty" jsonschema:"Exclude workers from being exported."`
	ExcludeWorkersRegex  *StringSliceArgs `json:"excludeWorkersRegex,omitempty" jsonschema:"Exclude workers from being exported based on a regex."`
	ExcludeWorkersExcept *StringSliceArgs `json:"excludeWorkersExcept,omitempty" jsonschema:"Exclude all workers except for those defined in this list. The targets in excludeWorkers take precedence, so a worker defined here and in excludeWorkers is excluded."`

	ExcludeAllGitCredentials *bool `json:"excludeAllGitCredentials,omitempty" jsonschema:"Exclude all git credentials. Must be used with -excludeCaCProjectSettings."`

	ExcludeAllDeploymentFreezes    *bool            `json:"excludeAllDeploymentFreezes,omitempty" jsonschema:"Exclude all deployment freezes from being exported."`
	ExcludeDeploymentFreezes       *StringSliceArgs `json:"excludeDeploymentFreezes,omitempty" jsonschema:"Exclude a deployment freeze from being exported."`
	ExcludeDeploymentFreezesExcept *StringSliceArgs `json:"excludeDeploymentFreezesExcept,omitempty" jsonschema:"All deployment freezes except those defined with excludeDeploymentFreezesExcept are excluded."`
	ExcludeDeploymentFreezesRegex  *StringSliceArgs `json:"excludeDeploymentFreezesRegex,omitempty" jsonschema:"Exclude a deployment freeze from being exported based on regex match."`

	ExcludePlatformHubVersionControl *bool `json:"excludePlatformHubVersionControl,omitempty" jsonschema:"Exclude the Platform Hub version control settings."`
}

// GetBackend forces the use of a local backend for stateless exports
func (arguments *Arguments) GetBackend() string {
	if *arguments.Stateless {
		return ""
	}

	return *arguments.BackendBlock
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

	// Initialize all pointer fields
	arguments.InsecureTls = new(bool)
	arguments.ExperimentalEnableStepTemplates = new(bool)
	arguments.Profiling = new(bool)
	arguments.ExcludeTerraformVariables = new(bool)
	arguments.ExcludeSpaceCreation = new(bool)
	arguments.ConfigFile = new(string)
	arguments.ConfigPath = new(string)
	arguments.Version = new(bool)
	arguments.IgnoreInvalidExcludeExcept = new(bool)
	arguments.Url = new(string)
	arguments.ApiKey = new(string)
	arguments.AccessToken = new(string)
	arguments.UseRedirector = new(bool)
	arguments.RedirectorHost = new(string)
	arguments.RedirectorServiceApiKey = new(string)
	arguments.RedirecrtorApiKey = new(string)
	arguments.RedirectorRedirections = new(string)
	arguments.Space = new(string)
	arguments.Destination = new(string)
	arguments.Console = new(bool)
	arguments.ProjectId = &StringSliceArgs{}
	arguments.ProjectName = &StringSliceArgs{}
	arguments.RunbookId = new(string)
	arguments.RunbookName = new(string)
	arguments.LookupProjectDependencies = new(bool)
	arguments.LookupProjectLinkTenants = new(bool)
	arguments.Stateless = new(bool)
	arguments.StatelessAdditionalParams = &StringSliceArgs{}
	arguments.StepTemplateName = new(string)
	arguments.StepTemplateKey = new(string)
	arguments.StepTemplateDescription = new(string)
	arguments.IgnoreCacManagedValues = new(bool)
	arguments.ExcludeCaCProjectSettings = new(bool)
	arguments.BackendBlock = new(string)
	arguments.DetachProjectTemplates = new(bool)
	arguments.DefaultSecretVariableValues = new(bool)
	arguments.DummySecretVariableValues = new(bool)
	arguments.InlineVariableValues = new(bool)
	arguments.ProviderVersion = new(string)
	arguments.ExcludeProvider = new(bool)
	arguments.IncludeProviderServerDetails = new(bool)
	arguments.IncludeOctopusOutputVars = new(bool)
	arguments.LimitAttributeLength = new(int)
	arguments.LimitResourceCount = new(int)
	arguments.GenerateImportScripts = new(bool)
	arguments.IgnoreCacErrors = new(bool)
	arguments.OctopusManagedTerraformVars = new(string)
	arguments.IgnoreProjectChanges = new(bool)
	arguments.IgnoreProjectVariableChanges = new(bool)
	arguments.IgnoreProjectGroupChanges = new(bool)
	arguments.IgnoreProjectNameChanges = new(bool)
	arguments.LookUpDefaultWorkerPools = new(bool)
	arguments.IncludeIds = new(bool)
	arguments.IncludeSpaceInPopulation = new(bool)
	arguments.IncludeDefaultChannel = new(bool)
	arguments.ExcludeAllProjectVariables = new(bool)
	arguments.ExcludeProjectVariables = &StringSliceArgs{}
	arguments.ExcludeProjectVariablesExcept = &StringSliceArgs{}
	arguments.ExcludeProjectVariablesRegex = &StringSliceArgs{}
	arguments.ExcludeVariableEnvironmentScopes = &StringSliceArgs{}
	arguments.ExcludeAllTenantVariables = new(bool)
	arguments.ExcludeTenantVariables = &StringSliceArgs{}
	arguments.ExcludeTenantVariablesExcept = &StringSliceArgs{}
	arguments.ExcludeTenantVariablesRegex = &StringSliceArgs{}
	arguments.ExcludeAllSteps = new(bool)
	arguments.ExcludeSteps = &StringSliceArgs{}
	arguments.ExcludeStepsRegex = &StringSliceArgs{}
	arguments.ExcludeStepsExcept = &StringSliceArgs{}
	arguments.ExcludeAllChannels = new(bool)
	arguments.ExcludeChannels = &StringSliceArgs{}
	arguments.ExcludeChannelsRegex = &StringSliceArgs{}
	arguments.ExcludeChannelsExcept = &StringSliceArgs{}
	arguments.ExcludeInvalidChannels = new(bool)
	arguments.ExcludeAllRunbooks = new(bool)
	arguments.ExcludeRunbooks = &StringSliceArgs{}
	arguments.ExcludeRunbooksRegex = &StringSliceArgs{}
	arguments.ExcludeRunbooksExcept = &StringSliceArgs{}
	arguments.ExcludeAllTriggers = new(bool)
	arguments.ExcludeTriggers = &StringSliceArgs{}
	arguments.ExcludeTriggersRegex = &StringSliceArgs{}
	arguments.ExcludeTriggersExcept = &StringSliceArgs{}
	arguments.ExcludeLibraryVariableSets = &StringSliceArgs{}
	arguments.ExcludeLibraryVariableSetsRegex = &StringSliceArgs{}
	arguments.ExcludeLibraryVariableSetsExcept = &StringSliceArgs{}
	arguments.ExcludeAllLibraryVariableSets = new(bool)
	arguments.ExcludeEnvironments = &StringSliceArgs{}
	arguments.ExcludeEnvironmentsRegex = &StringSliceArgs{}
	arguments.ExcludeEnvironmentsExcept = &StringSliceArgs{}
	arguments.ExcludeAllEnvironments = new(bool)
	arguments.ExcludeFeeds = &StringSliceArgs{}
	arguments.ExcludeFeedsRegex = &StringSliceArgs{}
	arguments.ExcludeFeedsExcept = &StringSliceArgs{}
	arguments.ExcludeAllFeeds = new(bool)
	arguments.ExcludeProjectGroups = &StringSliceArgs{}
	arguments.ExcludeProjectGroupsRegex = &StringSliceArgs{}
	arguments.ExcludeProjectGroupsExcept = &StringSliceArgs{}
	arguments.ExcludeAllProjectGroups = new(bool)
	arguments.ExcludeAccounts = &StringSliceArgs{}
	arguments.ExcludeAccountsRegex = &StringSliceArgs{}
	arguments.ExcludeAccountsExcept = &StringSliceArgs{}
	arguments.ExcludeAllAccounts = new(bool)
	arguments.ExcludeCertificates = &StringSliceArgs{}
	arguments.ExcludeCertificatesRegex = &StringSliceArgs{}
	arguments.ExcludeCertificatesExcept = &StringSliceArgs{}
	arguments.ExcludeAllCertificates = new(bool)
	arguments.ExcludeLifecycles = &StringSliceArgs{}
	arguments.ExcludeLifecyclesRegex = &StringSliceArgs{}
	arguments.ExcludeLifecyclesExcept = &StringSliceArgs{}
	arguments.ExcludeAllLifecycles = new(bool)
	arguments.ExcludeWorkerpools = &StringSliceArgs{}
	arguments.ExcludeWorkerpoolsRegex = &StringSliceArgs{}
	arguments.ExcludeWorkerpoolsExcept = &StringSliceArgs{}
	arguments.ExcludeAllWorkerpools = new(bool)
	arguments.ExcludeMachinePolicies = &StringSliceArgs{}
	arguments.ExcludeMachinePoliciesRegex = &StringSliceArgs{}
	arguments.ExcludeMachinePoliciesExcept = &StringSliceArgs{}
	arguments.ExcludeAllMachinePolicies = new(bool)
	arguments.ExcludeMachineProxies = &StringSliceArgs{}
	arguments.ExcludeMachineProxiesRegex = &StringSliceArgs{}
	arguments.ExcludeMachineProxiesExcept = &StringSliceArgs{}
	arguments.ExcludeAllMachineProxies = new(bool)
	arguments.ExcludeTenantTags = &StringSliceArgs{}
	arguments.ExcludeTenantTagSets = &StringSliceArgs{}
	arguments.ExcludeTenantTagSetsRegex = &StringSliceArgs{}
	arguments.ExcludeTenantTagSetsExcept = &StringSliceArgs{}
	arguments.ExcludeAllTenantTagSets = new(bool)
	arguments.ExcludeTenants = &StringSliceArgs{}
	arguments.ExcludeTenantsRegex = &StringSliceArgs{}
	arguments.ExcludeTenantsWithTags = &StringSliceArgs{}
	arguments.ExcludeTenantsExcept = &StringSliceArgs{}
	arguments.ExcludeAllTenants = new(bool)
	arguments.ExcludeProjects = &StringSliceArgs{}
	arguments.ExcludeProjectsExcept = &StringSliceArgs{}
	arguments.ExcludeProjectsRegex = &StringSliceArgs{}
	arguments.ExcludeAllProjects = new(bool)
	arguments.ExcludeAllTargets = new(bool)
	arguments.ExcludeTargets = &StringSliceArgs{}
	arguments.ExcludeTargetsRegex = &StringSliceArgs{}
	arguments.ExcludeTargetsExcept = &StringSliceArgs{}
	arguments.ExcludeTargetsWithNoEnvironments = new(bool)
	arguments.ExcludeAllWorkers = new(bool)
	arguments.ExcludeWorkers = &StringSliceArgs{}
	arguments.ExcludeWorkersRegex = &StringSliceArgs{}
	arguments.ExcludeWorkersExcept = &StringSliceArgs{}
	arguments.ExcludeAllGitCredentials = new(bool)
	arguments.ExcludeAllDeploymentFreezes = new(bool)
	arguments.ExcludeDeploymentFreezes = &StringSliceArgs{}
	arguments.ExcludeDeploymentFreezesExcept = &StringSliceArgs{}
	arguments.ExcludeDeploymentFreezesRegex = &StringSliceArgs{}
	arguments.ExcludePlatformHubVersionControl = new(bool)

	flags.StringVar(arguments.ConfigFile, "configFile", "octoterra", "The name of the configuration file to use. Do not include the extension. Defaults to octoterra")
	flags.StringVar(arguments.ConfigPath, "configPath", ".", "The path of the configuration file to use. Defaults to the current directory")
	flags.IntVar(arguments.LimitAttributeLength, "limitAttributeLength", 0, "For internal use only. Limits the length of the attribute names.")
	flags.IntVar(arguments.LimitResourceCount, "limitResourceCount", 0, "For internal use only. Limits the number of resources of a given type that are returned. For example, a value of 30 will ensure the exported Terraform only includes up to 30 accounts, and up to 30 feeds, and up to 30 projects etc. This is used to reduce the output when octoterra is used to generate a context for an LLM. This limit is a guide and it is possible that more than the specified number of resources are returned due to multiple goroutines adding resources to the output.")
	flags.BoolVar(arguments.Profiling, "profiling", false, "Enable profiling. Run \"pprof -http=:8080 octoterra.prof\" to view the results.")
	flags.BoolVar(arguments.IgnoreCacErrors, "ignoreCacErrors", false, "Ignores errors that would arise when a project can not resolve configuration in a Git repo.")
	flags.BoolVar(arguments.ExperimentalEnableStepTemplates, "experimentalEnableStepTemplates", false, "Has no effect. This option used to enable the export of step templates, but this is now a standard feature. This option is left in for compatibility.")
	flags.BoolVar(arguments.ExcludeTerraformVariables, "excludeTerraformVariables", false, "This option means the exported module does not expose Terraform variables for common inputs like the value of project or library variables set variables. This reduces the size of the Terraform configuration files, but makes the module less configurable because values are hard coded.")
	flags.BoolVar(arguments.ExcludeSpaceCreation, "excludeSpaceCreation", false, "This option excludes the Terraform configuration that is used to create the space.")
	flags.BoolVar(arguments.IgnoreInvalidExcludeExcept, "ignoreInvalidExcludeExcept", false, "Ensures that resource names passed to the 'Exclude<ResourceType>Except' arguments are valid, and if they are not, removes those names from the list. This is useful when an external system attempts to filter results but places incorrect values into 'Exclude<ResourceType>Except' arguments. It may result in all resources being returned if no valid resources names are included in the 'Exclude<ResourceType>Except' arguments.")
	flags.BoolVar(arguments.Version, "version", false, "Print the version")
	flags.BoolVar(arguments.IncludeIds, "includeIds", false, "For internal use only. Include the \"id\" field on generated resources. Note that this is almost always unnecessary and undesirable.")
	flags.BoolVar(arguments.IncludeSpaceInPopulation, "includeSpaceInPopulation", false, "For internal use only. Include the space resource in the space population script. Note that this is almost always unnecessary and undesirable, as the space resources are included in the space creation module.")
	flags.BoolVar(arguments.GenerateImportScripts, "generateImportScripts", false, "Generate Bash and Powershell scripts used to import resources into the Terraform state.")
	flags.BoolVar(arguments.InsecureTls, "insecureTls", false, "Ignore certificate errors when connecting to the Octopus server.")
	flags.StringVar(arguments.Url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app - this is also defined in the OCTOPUS_CLI_SERVER environment variable")
	flags.StringVar(arguments.Space, "space", "", "The Octopus space name or ID")
	flags.StringVar(arguments.ApiKey, "apiKey", "", "The Octopus api key - this is also defined in the OCTOPUS_CLI_API_KEY environment variable")
	flags.StringVar(arguments.AccessToken, "accessToken", "", "The Octopus access token")
	flags.StringVar(arguments.Destination, "dest", "", "The directory to place the Terraform files in")
	flags.BoolVar(arguments.Console, "console", false, "Dump Terraform files to the console")

	flags.BoolVar(arguments.UseRedirector, "useRedirector", false, "Set to true to access the Octopus instance via the redirector")
	flags.StringVar(arguments.RedirectorHost, "redirectorHost", "", "The hostname of the redirector service")
	flags.StringVar(arguments.RedirectorServiceApiKey, "redirectorServiceApiKey", "", "The service api key of the redirector service")
	flags.StringVar(arguments.RedirecrtorApiKey, "redirecrtorApiKey", "", "The user api key of the redirector service")
	flags.StringVar(arguments.RedirectorRedirections, "redirectorRedirections", "", "The redirection rules for the redirector service")

	flags.BoolVar(arguments.Stateless, "stepTemplate", false, "Create an Octopus step template")
	flags.Var(arguments.StatelessAdditionalParams, "stepTemplateAdditionalParameters", "Indicates that a non-secret variable should be exposed as a parameter. The format of this option is \"ProjectName:VariableName\". This option is only used with the -stepTemplate option.")
	flags.StringVar(arguments.StepTemplateName, "stepTemplateName", "", "Step template name. Only used with the stepTemplate option.")
	flags.StringVar(arguments.StepTemplateKey, "stepTemplateKey", "", "Step template key used when building parameter names. Only used with the stepTemplate option.")
	flags.StringVar(arguments.StepTemplateDescription, "stepTemplateDescription", "", "Step template description used when building parameter names. Only used with the stepTemplate option.")
	flags.Var(arguments.ProjectId, "projectId", "Limit the export to a single project")
	flags.Var(arguments.ProjectName, "projectName", "Limit the export to a single project")
	flags.StringVar(arguments.RunbookId, "runbookId", "", "Limit the export to a single runbook. Runbooks are exported referencing external resources as data sources.")
	flags.StringVar(arguments.RunbookName, "runbookName", "", "Limit the export to a single runbook. Requires projectName or projectId. Runbooks are exported referencing external resources as data sources.")
	flags.BoolVar(arguments.LookupProjectDependencies, "lookupProjectDependencies", false, "Use data sources to lookup the external project dependencies. Use this when the destination space has existing environments, accounts, tenants, feeds, git credentials, and library variable sets that this project should reference.")
	flags.BoolVar(arguments.LookupProjectLinkTenants, "lookupProjectLinkTenants", false, "When lookupProjectDependencies is true, lookupProjectLinkTenants will reestablish the link to tenants that were linked to the source project and recreate any project and common tenant variables. Essentially this means the exported project \"owns\" the relationship to the tenant and any variables used by the tenant.")
	flags.BoolVar(arguments.IgnoreCacManagedValues, "ignoreCacManagedValues", true, "Pass this to exclude values managed by Config-as-Code from the exported Terraform. This includes non-sensitive variables, the deployment process, connectivity settings, and other project settings. This has no effect on projects that do not have CaC enabled.")
	flags.BoolVar(arguments.ExcludeCaCProjectSettings, "excludeCaCProjectSettings", false, "Pass this to exclude any Config-As-Code settings in the exported projects. Typically you set -ignoreCacManagedValues=false -excludeCaCProjectSettings=true to essentially \"convert\" a CaC project to a regular project. Values from the \"main\" or \"master\" branches will be used first, or just fall back to the first configured branch.")
	flags.BoolVar(arguments.DefaultSecretVariableValues, "defaultSecretVariableValues", false, "Pass this to set the default value of secret variables to the octostache template referencing the variable.")
	flags.BoolVar(arguments.DummySecretVariableValues, "dummySecretVariableValues", false, "Pass this to set the default value of secret variables, account secrets, feed credentials to a dummy value. This allows resources with secret values to be created without knowing the secrets, while still allowing the secret values to be specified if they are known. This option takes precedence over the defaultSecretVariableValues option.")
	flags.BoolVar(arguments.InlineVariableValues, "inlineVariableValues", false, "Inline the project and library variable set variable values rather than exposing their value as a Terraform variable. Secret variables will be inlined as dummy values. This option takes precedence over DummySecretVariableValues and DefaultSecretVariableValues.")
	flags.StringVar(arguments.BackendBlock, "terraformBackend", "", "Specifies the backend type to be added to the exported Terraform configuration.")
	flags.StringVar(arguments.ProviderVersion, "providerVersion", "", "Specifies the Octopus Terraform provider version.")
	flags.StringVar(arguments.OctopusManagedTerraformVars, "octopusManagedTerraformVars", "", "Specifies the name of an Octopus variable to be used as a template string in the body of the terraform.tfvars file. This allows Octopus to inject all the variables used by Terraform from a variable containing the contents of a terraform.tfvars file.")
	flags.BoolVar(arguments.DetachProjectTemplates, "detachProjectTemplates", false, "Detaches any step templates in the exported Terraform.")
	flags.BoolVar(arguments.IncludeDefaultChannel, "includeDefaultChannel", false, "Internal use only. Includes the \"Default\" channel as a standard channel resource rather than a data block.")

	flags.BoolVar(arguments.ExcludeAllSteps, "excludeAllSteps", false, "Exclude all steps when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(arguments.ExcludeSteps, "excludeSteps", "A steps to be excluded when exporting projects or runbooks. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(arguments.ExcludeStepsRegex, "excludeStepsRegex", "A step to be excluded when exporting projects or runbooks based on regex match. WARNING: variables scoped to this step will no longer have the step scope applied.")
	flags.Var(arguments.ExcludeStepsExcept, "excludeStepsExcept", "All step except those defined with excludeRunbooksExcept are excluded when exporting a project or runbook. WARNING: variables scoped to other step will no longer have the step scope applied.")

	flags.BoolVar(arguments.ExcludeAllRunbooks, "excludeAllRunbooks", false, "Exclude all runbooks when exporting a project or space. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(arguments.ExcludeRunbooks, "excludeRunbook", "A runbook to be excluded when exporting a single project. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(arguments.ExcludeRunbooksRegex, "excludeRunbookRegex", "A runbook to be excluded when exporting a single project based on regex match. WARNING: variables scoped to this runbook will no longer have the runbook scope applied.")
	flags.Var(arguments.ExcludeRunbooksExcept, "excludeRunbooksExcept", "All runbooks except those defined with excludeRunbooksExcept are excluded when exporting a single project. WARNING: variables scoped to other runbooks will no longer have the runbook scope applied.")

	flags.BoolVar(arguments.ExcludeAllTriggers, "excludeAllTriggers", false, "Exclude all triggers when exporting a project or space.")
	flags.Var(arguments.ExcludeTriggers, "excludeTrigger", "A trigger to be excluded when exporting a single project.")
	flags.Var(arguments.ExcludeTriggersRegex, "excludeTriggerRegex", "A trigger to be excluded when exporting a single project based on regex match.")
	flags.Var(arguments.ExcludeTriggersExcept, "excludeTriggersExcept", "All triggers except those defined with excludeTriggersExcept are excluded when exporting a single project.")

	flags.BoolVar(arguments.ExcludeAllLibraryVariableSets, "excludeAllLibraryVariableSets", false, "Exclude all library variable sets. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(arguments.ExcludeLibraryVariableSets, "excludeLibraryVariableSet", "A library variable set to be excluded when exporting a single project. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(arguments.ExcludeLibraryVariableSetsRegex, "excludeLibraryVariableSetRegex", "A library variable set to be excluded when exporting a single project based on regex match. WARNING: projects that linked this library variable set will no longer include these variables.")
	flags.Var(arguments.ExcludeLibraryVariableSetsExcept, "excludeLibraryVariableSetsExcept", "All library variable sets except those defined with excludeAllLibraryVariableSets are excluded. WARNING: projects that linked other library variable set will no longer include these variables.")

	flags.BoolVar(arguments.ExcludeAllEnvironments, "excludeAllEnvironments", false, "Exclude all environments.  WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeEnvironments, "excludeEnvironments", "An environment to be excluded when exporting a single project. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeEnvironmentsRegex, "excludeEnvironmentsRegex", "A environment to be excluded when exporting a single project based on regex match. WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeEnvironmentsExcept, "excludeEnvironmentsExcept", "All environments except those defined with excludeEnvironmentsExcept are excluded.  WARNING: this can have unexpected side effects, such as variables becoming unscoped. The exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllFeeds, "excludeAllFeeds", false, "Exclude all feeds.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeFeeds, "excludeFeeds", "A feed to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeFeedsRegex, "excludeFeedsRegex", "A feed to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeFeedsExcept, "excludeFeedsExcept", "All feeds except those defined with excludeFeedsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllAccounts, "excludeAllAccounts", false, "Exclude all Accounts.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeAccounts, "excludeAccounts", "A feed to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeAccountsRegex, "excludeAccountsRegex", "An account to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeAccountsExcept, "excludeAccountsExcept", "All accounts except those defined with excludeAccountsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllProjectGroups, "excludeAllProjectGroups", false, "Exclude all project groups.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeProjectGroups, "excludeProjectGroups", "A project group to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeProjectGroupsRegex, "excludeProjectGroupsRegex", "A project group to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeProjectGroupsExcept, "excludeProjectGroupsExcept", "All project groups except those defined with excludeProjectGroupsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllCertificates, "excludeAllCertificates", false, "Exclude all Certificates.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeCertificates, "excludeCertificates", "A certificate to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeCertificatesRegex, "excludeCertificatesRegex", "A certificate to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeCertificatesExcept, "excludeCertificatesExcept", "All Certificates except those defined with excludeCertificatesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllLifecycles, "excludeAllLifecycles", false, "Exclude all lifecycles.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeLifecycles, "excludeLifecycles", "A lifecycle to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeLifecyclesRegex, "excludeLifecyclesRegex", "A lifecycle to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeLifecyclesExcept, "excludeLifecyclesExcept", "All lifecycles except those defined with excludeLifecyclesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllWorkerpools, "excludeAllWorkerPools", false, "Exclude all worker pools.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeWorkerpools, "excludeWorkerPools", "A worker pool to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeWorkerpoolsRegex, "excludeWorkerPoolsRegex", "A worker pool to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeWorkerpoolsExcept, "excludeWorkerPoolsExcept", "All worker pools except those defined with excludeWorkerpoolsExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllMachinePolicies, "excludeAllMachinePolicies", false, "Exclude all machine policies.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeMachinePolicies, "excludeMachinePolicies", "A machine policy to be excluded when exporting a single project. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeMachinePoliciesRegex, "excludeMachinePoliciesRegex", "A machine policy to be excluded when exporting a single project based on regex match. WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")
	flags.Var(arguments.ExcludeMachinePoliciesExcept, "excludeMachinePoliciesExcept", "All machine policies except those defined with excludeMachinePoliciesExcept are excluded.  WARNING: the exported module is unlikely to be complete and will fail to apply if this option is enabled.")

	flags.BoolVar(arguments.ExcludeAllMachineProxies, "excludeAllMachineProxies", false, "Exclude all machine proxies.")
	flags.Var(arguments.ExcludeMachineProxies, "excludeMachineProxies", "A machine proxy to be excluded when exporting a single project.")
	flags.Var(arguments.ExcludeMachineProxiesRegex, "excludeMachineProxiesRegex", "A machine proxy to be excluded when exporting a single project based on regex match.")
	flags.Var(arguments.ExcludeMachineProxiesExcept, "excludeMachineProxiesExcept", "All machine proxies except those defined with excludeMachineProxiesExcept are excluded.")

	flags.BoolVar(arguments.ExcludeAllTenantVariables, "excludeAllTenantVariables", false, "Exclude all tenant variables from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(arguments.ExcludeTenantVariables, "excludeTenantVariables", "Exclude a tenant variable from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(arguments.ExcludeTenantVariablesExcept, "excludeTenantVariablesExcept", "All tenant variables except those defined with excludeTenantVariablesExcept are excluded. WARNING: steps that used other variables may no longer function correctly.")
	flags.Var(arguments.ExcludeTenantVariablesRegex, "excludeTenantVariablesRegex", "Exclude a tenant variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly.")

	flags.BoolVar(arguments.ExcludeAllProjectVariables, "excludeAllProjectVariables", false, "Exclude all project variables from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(arguments.ExcludeProjectVariables, "excludeProjectVariable", "Exclude a project variable from being exported. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(arguments.ExcludeProjectVariablesExcept, "excludeProjectVariableExcept", "All project variables except those defined with excludeProjectVariableExcept are excluded. WARNING: steps that used other variables may no longer function correctly.")
	flags.Var(arguments.ExcludeProjectVariablesRegex, "excludeProjectVariableRegex", "Exclude a project variable from being exported based on regex match. WARNING: steps that used this variable may no longer function correctly.")
	flags.Var(arguments.ExcludeVariableEnvironmentScopes, "excludeVariableEnvironmentScopes", "Exclude a environment when it appears in a variable's environment scope. WARNING: variables scoped to this environment will no longer have that environment scope applied.")

	// missing all, regex, except
	flags.Var(arguments.ExcludeTenantTags, "excludeTenantTags", "Exclude an individual tenant tag from being exported. Tags are in the format \"taggroup/tagname\". WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied.")

	flags.Var(arguments.ExcludeTenantTagSets, "excludeTenantTagSets", "Exclude a tenant tag set from being exported. WARNING: Steps that were set to run on tenants with excluded tags will no longer have that condition applied.")
	flags.BoolVar(arguments.ExcludeAllTenantTagSets, "excludeAllTenantTagSets", false, "Exclude all tenant tag sets from being exported. WARNING: Resources scoped to tenant tag sets will be unscoped.")
	flags.Var(arguments.ExcludeTenantTagSetsRegex, "excludeTenantTagSetsRegex", "Exclude tenant tag sets from being exported based on a regex. WARNING: Resources scoped to tenant tag sets will be unscoped")
	flags.Var(arguments.ExcludeTenantTagSetsExcept, "excludeTenantTagSetsExcept", "Exclude all tenant tag sets except for those define in this list. WARNING: Resources scoped to tenant tag sets will be unscoped")

	flags.BoolVar(arguments.ExcludeAllTenants, "excludeAllTenants", false, "Exclude all tenants from being exported.")
	flags.Var(arguments.ExcludeTenants, "excludeTenants", "Exclude a tenant from being exported.")
	flags.Var(arguments.ExcludeTenantsRegex, "excludeTenantsRegex", "Exclude a tenant from being exported based on a regex.")
	flags.Var(arguments.ExcludeTenantsWithTags, "excludeTenantsWithTag", "Exclude any tenant with this tag from being exported. This is useful when using tags to separate tenants that can be exported with those that should not. Tags are in the format TagGroupName/TagName.")
	flags.Var(arguments.ExcludeTenantsExcept, "excludeTenantsExcept", "Exclude all tenants except for those define in this list. The tenants in excludeTenants take precedence, so a tenant define here and in excludeTenants is excluded.")

	flags.BoolVar(arguments.ExcludeAllTargets, "excludeAllTargets", false, "Exclude all targets from being exported. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.BoolVar(arguments.ExcludeTargetsWithNoEnvironments, "excludeTargetsWithNoEnvironments", false, "Exclude targets that have had all their environments excluded. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(arguments.ExcludeTargets, "excludeTargets", "Exclude targets from being exported. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(arguments.ExcludeTargetsRegex, "excludeTargetsRegex", "Exclude targets from being exported based on a regex. WARNING: Variables that were scoped to targets will become unscoped.")
	flags.Var(arguments.ExcludeTargetsExcept, "excludeTargetsExcept", "Exclude all targets except for those define in this list. The targets in excludeTargets take precedence, so a tenant define here and in excludeTargets is excluded. WARNING: Variables that were scoped to other targets will become unscoped.")

	flags.BoolVar(arguments.ExcludeAllWorkers, "excludeAllWorkers", false, "Exclude all workers from being exported.")
	flags.Var(arguments.ExcludeWorkers, "excludeWorkers", "Exclude workers from being exported.")
	flags.Var(arguments.ExcludeWorkersRegex, "excludeWorkersRegex", "Exclude workers from being exported based on a regex.")
	flags.Var(arguments.ExcludeWorkersExcept, "excludeWorkersExcept", "Exclude all workers except for those define in this list. The targets in excludeWorkers take precedence, so a worker define here and in excludeWorkers is excluded.")

	flags.BoolVar(arguments.ExcludeAllProjects, "excludeAllProjects", false, "Exclude all projects from being exported. This is only used when exporting a space.")
	flags.Var(arguments.ExcludeProjects, "excludeProjects", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.Var(arguments.ExcludeProjectsRegex, "excludeProjectsRegex", "Exclude a project from being exported. This is only used when exporting a space.")
	flags.Var(arguments.ExcludeProjectsExcept, "excludeProjectsExcept", "All projects except those defined with excludeProjectsExcept are excluded. This is only used when exporting a space.")

	flags.BoolVar(arguments.ExcludeAllChannels, "excludeAllChannels", false, "Exclude all channels from being exported. WARNING: Variables and steps that were scoped to channels will become unscoped.")
	flags.Var(arguments.ExcludeChannels, "excludeChannels", "Exclude a channel from being exported. WARNING: Variables and steps that were scoped to channels will become unscoped.")
	flags.Var(arguments.ExcludeChannelsRegex, "excludeChannelsRegex", "Exclude a channel from being exported. WARNING: Variables and steps that were scoped to channels will become unscoped.")
	flags.Var(arguments.ExcludeChannelsExcept, "excludeChannelsExcept", "All channels except those defined with excludeChannelsExcept are excluded. WARNING: Variables and steps that were scoped to channels will become unscoped.")
	flags.BoolVar(arguments.ExcludeInvalidChannels, "excludeInvalidChannels", false, "Channels that reference packages that are no longer defined in the deployment process will be excluded. WARNING: Variables and steps that were scoped to channels will become unscoped.")

	flags.BoolVar(arguments.ExcludeAllGitCredentials, "excludeAllGitCredentials", false, "Exclude all git credentials. Must be used with -excludeCaCProjectSettings.")

	flags.BoolVar(arguments.ExcludeAllDeploymentFreezes, "excludeAllDeploymentFreezes", false, "Exclude all deployment freezes from being exported.")
	flags.Var(arguments.ExcludeDeploymentFreezes, "excludeDeploymentFreezes", "Exclude a deployment freezes from being exported.")
	flags.Var(arguments.ExcludeDeploymentFreezesRegex, "excludeDeploymentFreezesRegex", "Exclude a deployment freezes from being exported.")
	flags.Var(arguments.ExcludeDeploymentFreezesExcept, "excludeDeploymentFreezesExcept", "All deployment freezes except those defined with excludeProjectsExcept are excluded.")

	flags.BoolVar(arguments.ExcludeProvider, "excludeProvider", false, "Exclude the provider from the exported Terraform configuration files. This is useful when you want to use a parent module to define the backend, as the parent module must define the provider.")
	flags.BoolVar(arguments.IncludeProviderServerDetails, "includeProviderServerDetails", true, "Define the server UL and API keys as variables passed to the provider. Set this to false to use the OCTOPUS_ACCESS_TOKEN, OCTOPUS_URL, and OCTOPUS_APIKEY environment variables to configure the provider.")

	flags.BoolVar(arguments.IncludeOctopusOutputVars, "includeOctopusOutputVars", true, "Capture the Octopus server URL, API key and Space ID as output variables. This is useful when querying the Terraform state file to locate where the resources were created.")
	flags.BoolVar(arguments.IgnoreProjectChanges, "ignoreProjectChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project (including its variables) when exporting a single project.")
	flags.BoolVar(arguments.IgnoreProjectVariableChanges, "ignoreProjectVariableChanges", false, "Use the Terraform lifecycle meta-argument to ignore all changes to the project's variables when exporting a single project. This differs from the ignoreProjectChanges option by only ignoring changes to variables while reapplying changes to all other project settings.")
	flags.BoolVar(arguments.IgnoreProjectGroupChanges, "ignoreProjectGroupChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's group.")
	flags.BoolVar(arguments.IgnoreProjectNameChanges, "ignoreProjectNameChanges", false, "Use the Terraform lifecycle meta-argument to ignore the changes to the project's name.")
	flags.BoolVar(arguments.LookUpDefaultWorkerPools, "lookUpDefaultWorkerPools", false, "Reference the worker pool by name when a step uses the default worker pool. This means exported projects do not inherit the default worker pool when they are applied in a new space.")

	flags.BoolVar(arguments.ExcludePlatformHubVersionControl, "excludePlatformHubVersionControl", false, "Exclude the Platform Hub version control settings.")

	err := flags.Parse(args)

	if err != nil {
		return Arguments{}, buf.String(), err
	}

	err = overrideArgs(flags, *arguments.ConfigPath, *arguments.ConfigFile)

	if err != nil {
		return Arguments{}, buf.String(), err
	}

	if *arguments.Url == "" {
		*arguments.Url = os.Getenv("OCTOPUS_CLI_SERVER")
	}

	if *arguments.ApiKey == "" && *arguments.AccessToken == "" {
		*arguments.ApiKey = os.Getenv("OCTOPUS_CLI_API_KEY")
	}

	if err := arguments.ValidateExcludeExceptArgs(); err != nil {
		return Arguments{}, "", err
	}

	if err := arguments.ConfigureGlobalSettings(); err != nil {
		return Arguments{}, "", err
	}

	return arguments, buf.String(), nil
}

func (arguments *Arguments) ConfigureGlobalSettings() error {
	if *arguments.InsecureTls {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return nil
}

// ValidateExcludeExceptArgs removes any resource named in an Exclude<ResourceType>Except argument that does not
// exist in the Octopus instance. This is mostly used when external systems attempt to filter the results but
// may place incorrect values into the Exclude<ResourceType>Except arguments.
func (arguments *Arguments) ValidateExcludeExceptArgs() (funcErr error) {
	if !*arguments.IgnoreInvalidExcludeExcept {
		return
	}

	octopusClient := client.OctopusApiClient{
		Url:                     *arguments.Url,
		ApiKey:                  *arguments.ApiKey,
		AccessToken:             *arguments.AccessToken,
		Space:                   *arguments.Space,
		Version:                 "",
		UseRedirector:           *arguments.UseRedirector,
		RedirectorHost:          *arguments.RedirectorHost,
		RedirectorServiceApiKey: *arguments.RedirectorServiceApiKey,
		RedirecrtorApiKey:       *arguments.RedirecrtorApiKey,
		RedirectorRedirections:  *arguments.RedirectorRedirections,
	}

	filteredProjects, err := filterNamedResource[octopus.Project](octopusClient, "Projects", []string(*arguments.ExcludeProjectsExcept))

	if err != nil {
		return err
	}

	*arguments.ExcludeProjectsExcept = StringSliceArgs(filteredProjects)

	filteredEnvironments, err := filterNamedResource[octopus.Environment](octopusClient, "Environments", []string(*arguments.ExcludeEnvironmentsExcept))

	if err != nil {
		return err
	}

	*arguments.ExcludeEnvironmentsExcept = StringSliceArgs(filteredEnvironments)

	filteredTenants, err := filterNamedResource[octopus.Tenant](octopusClient, "Tenants", []string(*arguments.ExcludeTenantsExcept))

	if err != nil {
		return nil
	}

	*arguments.ExcludeTenantsExcept = StringSliceArgs(filteredTenants)

	filteredMachines, err := filterNamedResource[octopus.Machine](octopusClient, "Machines", []string(*arguments.ExcludeTargetsExcept))

	if err != nil {
		return err
	}

	*arguments.ExcludeTargetsExcept = StringSliceArgs(filteredMachines)

	filteredRunbooks, err := filterNamedResource[octopus.Runbook](octopusClient, "Runbooks", []string(*arguments.ExcludeRunbooksExcept))

	if err != nil {
		return err
	}

	*arguments.ExcludeRunbooksExcept = StringSliceArgs(filteredRunbooks)

	filteredVariableSets, err := filterNamedResource[octopus.LibraryVariableSet](octopusClient, "LibraryVariableSets", []string(*arguments.ExcludeLibraryVariableSetsExcept))

	if err != nil {
		return err
	}

	*arguments.ExcludeLibraryVariableSetsExcept = StringSliceArgs(filteredVariableSets)

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
	v.SetEnvPrefix("octoterra")

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

			anyValue := v.Get(configName)

			if types.IsArrayOrSlice(anyValue) {
				for _, value := range v.GetStringSlice(configName) {
					err := flags.Set(allFlags.Name, value)
					funcError = errors.Join(funcError, err)
				}
			} else {
				err := flags.Set(allFlags.Name, v.GetString(configName))
				funcError = errors.Join(funcError, err)
			}
		}
	})

	return funcError
}
