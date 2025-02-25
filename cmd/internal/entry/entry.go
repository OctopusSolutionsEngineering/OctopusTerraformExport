package entry

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/collections"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/converters"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/generators"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
)

// Entry takes the arguments, exports the Octopus resources to HCL in strings and returns the strings mapped to file names.
func Entry(parseArgs args.Arguments, version string) (map[string]string, error) {

	if parseArgs.Profiling {
		f, err := os.Create("octoterra.prof")
		if err != nil {
			return nil, err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return nil, err
		}
		defer pprof.StopCPUProfile()
	}

	if len(parseArgs.ProjectName) != 0 {

		projectIds := []string{}

		for _, project := range parseArgs.ProjectName {
			projectId, err := ConvertProjectNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, project, version)

			if err != nil {
				return nil, err
			}

			projectIds = append(projectIds, projectId)
		}

		parseArgs.ProjectId = projectIds
	}

	if parseArgs.RunbookName != "" {
		runbookId, err := ConvertRunbookNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, parseArgs.ProjectId[0], parseArgs.RunbookName, version)

		if err != nil {
			return nil, err
		}

		parseArgs.RunbookId = runbookId
	}

	dependencies, err := getDependencies(parseArgs, version)

	if err != nil {
		return nil, err
	}

	if parseArgs.Stateless {
		templateGenerator := generators.StepTemplateGenerator{}
		templateContent, err := templateGenerator.Generate(dependencies, parseArgs.StepTemplateName, parseArgs.StepTemplateKey, parseArgs.StepTemplateDescription)

		if err != nil {
			return nil, err
		}

		return map[string]string{"step_template.json": string(templateContent[:])}, nil
	} else {
		files, err := ProcessResources(dependencies.Resources)

		if err != nil {
			return nil, err
		}

		dummyLogs := logDummyValues(dependencies)

		zap.L().Info(dummyLogs)
		files["dummy_values.txt"] = dummyLogs

		return files, nil
	}
}

func logDummyValues(dependencies *data.ResourceDetailsCollection) string {
	if len(dependencies.DummyVariables) == 0 {
		return ""
	}

	message := strutil.StripMultilineWhitespace(`The follow dummy values were defined for sensitive values and certificates.
		These values must be defined when applying the module, or manually updated after the module is applied`)

	for _, dummy := range dependencies.DummyVariables {
		message += "Dummy value defined for variable " + dummy.VariableName + " associated with " + dummy.ResourceType + " called " + dummy.ResourceName + "\n"
	}

	return message
}

func getDependencies(parseArgs args.Arguments, version string) (*data.ResourceDetailsCollection, error) {
	if parseArgs.RunbookId != "" {
		zap.L().Info("Exporting runbook " + parseArgs.RunbookId + " in space " + parseArgs.Space)
		files, err := ConvertRunbookToTerraform(parseArgs, version)
		if err != nil {
			return nil, err
		}
		return files, nil
	} else if len(parseArgs.ProjectId) != 0 {
		zap.L().Info("Exporting project(s) " + strings.Join(parseArgs.ProjectId, ", ") + " in space " + parseArgs.Space)
		files, err := ConvertProjectToTerraform(parseArgs, version)
		if err != nil {
			return nil, err
		}
		return files, nil
	} else {
		zap.L().Info("Exporting space " + parseArgs.Space)
		files, err := ConvertSpaceToTerraform(parseArgs, version)
		if err != nil {
			return nil, err
		}
		return files, nil
	}
}

func ConvertProjectNameToId(url string, space string, apiKey string, name string, version string) (string, error) {
	octopusClient := client.OctopusApiClient{
		Url:     url,
		Space:   space,
		ApiKey:  apiKey,
		Version: version,
	}

	collection := octopus.GeneralCollection[octopus.Project]{}
	err := octopusClient.GetAllResources("Projects", &collection, []string{"name", name})

	if err != nil {
		return "", err
	}

	if len(collection.Items) == 0 {
		return "", errors.New("failed to return any projects in space " + space +
			" - check the API key has permission to list projects")
	}

	for _, p := range collection.Items {
		if p.Name == name {
			return p.Id, nil
		}
	}

	return "", errors.New("did not find project with name " + name + " in space " + space)
}

func ConvertRunbookNameToId(url string, space string, apiKey string, projectId string, runbookName string, version string) (string, error) {
	octopusClient := client.OctopusApiClient{
		Url:     url,
		Space:   space,
		ApiKey:  apiKey,
		Version: version,
	}

	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := octopusClient.GetAllResources("Projects/"+projectId+"/runbooks", &collection)

	if err != nil {
		return "", err
	}

	if len(collection.Items) == 0 {
		return "", errors.New("failed to return any runbooks for the project " + projectId + " in space " + space +
			" - check the API key has permission to list runbooks")
	}

	for _, p := range collection.Items {
		if p.Name == runbookName {
			return p.Id, nil
		}
	}

	return "", errors.New("did not find runbook with name " + runbookName + " for the project " + projectId + " in space " + space)
}

func ConvertSpaceToTerraform(args args.Arguments, version string) (*data.ResourceDetailsCollection, error) {
	group := errgroup.Group{}
	group.SetLimit(10)

	octopusClient := client.OctopusApiClient{
		Url:     args.Url,
		Space:   args.Space,
		ApiKey:  args.ApiKey,
		Version: version,
	}

	dependencies := data.ResourceDetailsCollection{}

	dummySecretGenerator := dummy.DummySecret{}

	tenantCommonVariableProcessor := converters.TenantCommonVariableProcessor{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
	}

	tenantProjectVariableConverter := converters.TenantProjectVariableConverter{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
		DummySecretVariableValues:    args.DummySecretVariableValues,
		DummySecretGenerator:         dummySecretGenerator,
	}

	tenantProjectConverter := converters.TenantProjectConverter{
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		ErrGroup:                 &group,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		Excluder:                 converters.DefaultExcluder{},
		Client:                   &octopusClient,
	}

	stepTemplateConverter := converters.StepTemplateConverter{
		ErrGroup:                        &group,
		Client:                          &octopusClient,
		ExcludeAllStepTemplates:         false,
		ExcludeStepTemplates:            nil,
		ExcludeStepTemplatesRegex:       nil,
		ExcludeStepTemplatesExcept:      nil,
		Excluder:                        converters.DefaultExcluder{},
		LimitResourceCount:              0,
		GenerateImportScripts:           false,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
	}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.GetBackend(),
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	if !args.Stateless {
		converters.TerraformProviderGenerator{
			TerraformBackend:         args.GetBackend(),
			ProviderVersion:          args.ProviderVersion,
			ExcludeProvider:          args.ExcludeProvider,
			IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
		}.ToHcl("space_creation", false, &dependencies)
	}

	machinePolicyConverter := converters.MachinePolicyConverter{
		Client:                       &octopusClient,
		ErrGroup:                     &group,
		ExcludeMachinePolicies:       args.ExcludeMachinePolicies,
		ExcludeMachinePoliciesRegex:  args.ExcludeMachinePoliciesRegex,
		ExcludeMachinePoliciesExcept: args.ExcludeMachinePoliciesExcept,
		ExcludeAllMachinePolicies:    args.ExcludeAllMachinePolicies,
		Excluder:                     converters.DefaultExcluder{},
		LimitResourceCount:           args.LimitResourceCount,
		IncludeIds:                   args.IncludeIds,
		IncludeSpaceInPopulation:     args.IncludeSpaceInPopulation,
		GenerateImportScripts:        args.GenerateImportScripts,
	}
	environmentConverter := converters.EnvironmentConverter{
		Client:                    &octopusClient,
		ExcludeEnvironments:       args.ExcludeEnvironments,
		ExcludeAllEnvironments:    args.ExcludeAllEnvironments,
		ExcludeEnvironmentsExcept: args.ExcludeEnvironmentsExcept,
		ExcludeEnvironmentsRegex:  args.ExcludeEnvironmentsRegex,
		Excluder:                  converters.DefaultExcluder{},
		ErrGroup:                  &group,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                         &octopusClient,
		ExcludeTenants:                 args.ExcludeTenants,
		ExcludeTenantsWithTags:         args.ExcludeTenantsWithTags,
		ExcludeAllTenants:              args.ExcludeAllTenants,
		ExcludeTenantsExcept:           args.ExcludeTenantsExcept,
		Excluder:                       converters.DefaultExcluder{},
		DummySecretVariableValues:      args.DummySecretVariableValues,
		DummySecretGenerator:           dummySecretGenerator,
		ExcludeProjects:                args.ExcludeProjects,
		ExcludeProjectsRegex:           args.ExcludeProjectsRegex,
		ExcludeAllProjects:             args.ExcludeAllProjects,
		ExcludeProjectsExcept:          args.ExcludeProjectsExcept,
		ErrGroup:                       &group,
		ExcludeAllTenantVariables:      args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:         args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept:   args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:    args.ExcludeTenantVariablesRegex,
		TenantCommonVariableProcessor:  tenantCommonVariableProcessor,
		TenantProjectVariableConverter: tenantProjectVariableConverter,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:                     &octopusClient,
		Excluder:                   converters.DefaultExcluder{},
		ExcludeTenantTags:          args.ExcludeTenantTags,
		ExcludeTenantTagSets:       args.ExcludeTenantTagSets,
		ErrGroup:                   &group,
		ExcludeTenantTagSetsRegex:  args.ExcludeTenantTagSetsRegex,
		ExcludeTenantTagSetsExcept: args.ExcludeTenantTagSetsExcept,
		ExcludeAllTenantTagSets:    args.ExcludeAllTenantTagSets,
		LimitResourceCount:         args.LimitResourceCount,
		GenerateImportScripts:      args.GenerateImportScripts,
	}
	tenantConverter := converters.TenantConverter{
		Client:                   &octopusClient,
		TenantVariableConverter:  tenantVariableConverter,
		EnvironmentConverter:     environmentConverter,
		TagSetConverter:          &tagsetConverter,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		Excluder:                 converters.DefaultExcluder{},
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		TenantProjectConverter:   tenantProjectConverter,
	}
	accountConverter := converters.AccountConverter{
		Client:                    &octopusClient,
		EnvironmentConverter:      machinePolicyConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  &group,
		ExcludeAccounts:           args.ExcludeAccounts,
		ExcludeAccountsRegex:      args.ExcludeAccountsRegex,
		ExcludeAccountsExcept:     args.ExcludeAccountsExcept,
		ExcludeAllAccounts:        args.ExcludeAllAccounts,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	lifecycleConverter := converters.LifecycleConverter{
		Client:                   &octopusClient,
		EnvironmentConverter:     environmentConverter,
		ErrGroup:                 &group,
		ExcludeLifecycles:        args.ExcludeLifecycles,
		ExcludeLifecyclesRegex:   args.ExcludeLifecyclesRegex,
		ExcludeLifecyclesExcept:  args.ExcludeLifecyclesExcept,
		ExcludeAllLifecycles:     args.ExcludeAllLifecycles,
		Excluder:                 converters.DefaultExcluder{},
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
		ErrGroup:                  &group,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}
	channelConverter := converters.ChannelConverter{
		Client:                   &octopusClient,
		LifecycleConverter:       lifecycleConverter,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		Excluder:                 converters.DefaultExcluder{},
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeDefaultChannel:    args.IncludeDefaultChannel,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IgnoreCacErrors:          args.IgnoreCacErrors,
		ExcludeAllChannels:       args.ExcludeAllChannels,
		ExcludeChannels:          args.ExcludeChannels,
		ExcludeChannelsRegex:     args.ExcludeChannelsRegex,
		ExcludeChannelsExcept:    args.ExcludeChannelsExcept,
		ExcludeInvalidChannels:   args.ExcludeInvalidChannels,
	}

	projectGroupConverter := converters.ProjectGroupConverter{
		Client:                     &octopusClient,
		ErrGroup:                   &group,
		ExcludeProjectGroups:       args.ExcludeProjectGroups,
		ExcludeProjectGroupsRegex:  args.ExcludeProjectGroupsRegex,
		ExcludeProjectGroupsExcept: args.ExcludeProjectGroupsExcept,
		ExcludeAllProjectGroups:    args.ExcludeAllProjectGroups,
		Excluder:                   converters.DefaultExcluder{},
		LimitResourceCount:         args.LimitResourceCount,
		IncludeIds:                 args.IncludeIds,
		IncludeSpaceInPopulation:   args.IncludeSpaceInPopulation,
		GenerateImportScripts:      args.GenerateImportScripts,
	}

	certificateConverter := converters.CertificateConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  &group,
		ExcludeCertificates:       args.ExcludeCertificates,
		ExcludeCertificatesRegex:  args.ExcludeCertificatesRegex,
		ExcludeCertificatesExcept: args.ExcludeCertificatesExcept,
		ExcludeAllCertificates:    args.ExcludeAllCertificates,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeIds:                args.IncludeIds,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	listeningWorkerConverter := converters.ListeningWorkerConverter{
		BaseWorkerConverter: converters.BaseWorkerConverter{
			Client:                   &octopusClient,
			ErrGroup:                 &group,
			Excluder:                 converters.DefaultExcluder{},
			MachinePolicyConverter:   machinePolicyConverter,
			ExcludeAllWorkers:        args.ExcludeAllWorkers,
			ExcludeWorkers:           args.ExcludeWorkers,
			ExcludeWorkersRegex:      args.ExcludeWorkersRegex,
			ExcludeWorkersExcept:     args.ExcludeWorkersExcept,
			LimitResourceCount:       args.LimitResourceCount,
			IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
			IncludeIds:               args.IncludeIds,
			GenerateImportScripts:    args.GenerateImportScripts,
		},
	}

	k8sAgentWorkerConverter := converters.KubernetesAgentWorkerConverter{
		BaseWorkerConverter: converters.BaseWorkerConverter{
			Client:                   &octopusClient,
			ErrGroup:                 &group,
			Excluder:                 converters.DefaultExcluder{},
			MachinePolicyConverter:   machinePolicyConverter,
			ExcludeAllWorkers:        args.ExcludeAllWorkers,
			ExcludeWorkers:           args.ExcludeWorkers,
			ExcludeWorkersRegex:      args.ExcludeWorkersRegex,
			ExcludeWorkersExcept:     args.ExcludeWorkersExcept,
			LimitResourceCount:       args.LimitResourceCount,
			IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
			IncludeIds:               args.IncludeIds,
			GenerateImportScripts:    args.GenerateImportScripts,
		},
	}

	workerPoolConverter := converters.WorkerPoolConverter{
		Client:                   &octopusClient,
		ErrGroup:                 &group,
		ExcludeWorkerpools:       args.ExcludeWorkerpools,
		ExcludeWorkerpoolsRegex:  args.ExcludeWorkerpoolsRegex,
		ExcludeWorkerpoolsExcept: args.ExcludeWorkerpoolsExcept,
		ExcludeAllWorkerpools:    args.ExcludeAllWorkerpools,
		Excluder:                 converters.DefaultExcluder{},
		LimitResourceCount:       args.LimitResourceCount,
		IncludeIds:               args.IncludeIds,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	feedConverter := converters.FeedConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ErrGroup:                  &group,
		ExcludeFeeds:              args.ExcludeFeeds,
		ExcludeFeedsRegex:         args.ExcludeFeedsRegex,
		ExcludeFeedsExcept:        args.ExcludeFeedsExcept,
		ExcludeAllFeeds:           args.ExcludeAllFeeds,
		Excluder:                  converters.DefaultExcluder{},
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		CertificateConverter:     certificateConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeIds:               args.IncludeIds,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	sshTargetConverter := converters.SshTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},

		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,

		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		ErrGroup:                  &group,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ErrGroup:                  &group,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ErrGroup:                 &group,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            &octopusClient,
		ChannelConverter:                  channelConverter,
		EnvironmentConverter:              environmentConverter,
		TagSetConverter:                   &tagsetConverter,
		AzureCloudServiceTargetConverter:  azureCloudServiceTargetConverter,
		AzureServiceFabricTargetConverter: azureServiceFabricTargetConverter,
		AzureWebAppTargetConverter:        azureWebAppTargetConverter,
		CloudRegionTargetConverter:        cloudRegionTargetConverter,
		KubernetesTargetConverter:         kubernetesTargetConverter,
		ListeningTargetConverter:          listeningTargetConverter,
		OfflineDropTargetConverter:        offlineDropTargetConverter,
		PollingTargetConverter:            pollingTargetConverter,
		SshTargetConverter:                sshTargetConverter,
		AccountConverter:                  accountConverter,
		FeedConverter:                     feedConverter,
		CertificateConverter:              certificateConverter,
		WorkerPoolConverter:               workerPoolConverter,
		IgnoreCacManagedValues:            args.IgnoreCacManagedValues,
		DefaultSecretVariableValues:       args.DefaultSecretVariableValues,
		DummySecretVariableValues:         args.DummySecretVariableValues,
		ExcludeAllProjectVariables:        args.ExcludeAllProjectVariables,
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesExcept:     args.ExcludeProjectVariablesExcept,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeTenantTagSets:              args.ExcludeTenantTagSets,
		ExcludeTenantTags:                 args.ExcludeTenantTags,
		IgnoreProjectChanges:              args.IgnoreProjectChanges || args.IgnoreProjectVariableChanges,
		DummySecretGenerator:              dummySecretGenerator,
		Excluder:                          converters.DefaultExcluder{},
		ErrGroup:                          &group,
		ExcludeTerraformVariables:         args.ExcludeTerraformVariables,
		LimitAttributeLength:              args.LimitAttributeLength,
		StatelessAdditionalParams:         args.StatelessAdditionalParams,
		GenerateImportScripts:             args.GenerateImportScripts,
		EnvironmentFilter: converters.EnvironmentFilter{
			Client:                           &octopusClient,
			ExcludeVariableEnvironmentScopes: args.ExcludeVariableEnvironmentScopes,
		},
		IgnoreCacErrors: args.IgnoreCacErrors,
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                           &octopusClient,
		VariableSetConverter:             &variableSetConverter,
		Excluded:                         args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex:  args.ExcludeLibraryVariableSetsRegex,
		ExcludeLibraryVariableSetsExcept: args.ExcludeLibraryVariableSetsExcept,
		ExcludeAllLibraryVariableSets:    args.ExcludeAllLibraryVariableSets,
		DummySecretVariableValues:        args.DummySecretVariableValues,
		DummySecretGenerator:             dummySecretGenerator,
		Excluder:                         converters.DefaultExcluder{},
		ErrGroup:                         &group,
		LimitResourceCount:               args.LimitResourceCount,
		GenerateImportScripts:            args.GenerateImportScripts,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  &octopusClient,
		ErrGroup:                &group,
	}

	runbookConverter := converters.RunbookConverter{
		Client: &octopusClient,
		RunbookProcessConverter: &converters.RunbookProcessConverter{
			Client:                          &octopusClient,
			OctopusActionProcessor:          nil,
			IgnoreProjectChanges:            false,
			WorkerPoolProcessor:             workerPoolProcessor,
			ExcludeTenantTags:               args.ExcludeTenantTags,
			ExcludeTenantTagSets:            args.ExcludeTenantTagSets,
			Excluder:                        converters.DefaultExcluder{},
			TagSetConverter:                 &tagsetConverter,
			LimitAttributeLength:            args.LimitAttributeLength,
			ExcludeAllSteps:                 args.ExcludeAllSteps,
			ExcludeSteps:                    args.ExcludeSteps,
			ExcludeStepsRegex:               args.ExcludeStepsRegex,
			ExcludeStepsExcept:              args.ExcludeStepsExcept,
			IgnoreInvalidExcludeExcept:      args.IgnoreInvalidExcludeExcept,
			ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
			DummySecretGenerator:            dummySecretGenerator,
			DummySecretVariableValues:       args.DummySecretVariableValues,
		},
		EnvironmentConverter:     environmentConverter,
		ExcludedRunbooks:         nil,
		ExcludeRunbooksRegex:     nil,
		Excluder:                 converters.DefaultExcluder{},
		ExcludeRunbooksExcept:    nil,
		ExcludeAllRunbooks:       false,
		ProjectConverter:         nil,
		IgnoreProjectChanges:     false,
		ErrGroup:                 &group,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	projectConverter := &converters.ProjectConverter{
		Client:                      &octopusClient,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		TenantVariableConverter:     tenantVariableConverter,
		DeploymentProcessConverter: &converters.DeploymentProcessConverter{
			Client:                          &octopusClient,
			OctopusActionProcessor:          nil,
			IgnoreProjectChanges:            false,
			WorkerPoolProcessor:             workerPoolProcessor,
			ExcludeTenantTags:               args.ExcludeTenantTags,
			ExcludeTenantTagSets:            args.ExcludeTenantTagSets,
			Excluder:                        converters.DefaultExcluder{},
			TagSetConverter:                 &tagsetConverter,
			LimitAttributeLength:            args.LimitAttributeLength,
			ExcludeTerraformVariables:       args.ExcludeTerraformVariables,
			ExcludeAllSteps:                 args.ExcludeAllSteps,
			ExcludeSteps:                    args.ExcludeSteps,
			ExcludeStepsRegex:               args.ExcludeStepsRegex,
			ExcludeStepsExcept:              args.ExcludeStepsExcept,
			IgnoreInvalidExcludeExcept:      args.IgnoreInvalidExcludeExcept,
			ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
			DummySecretGenerator:            dummySecretGenerator,
			DummySecretVariableValues:       args.DummySecretVariableValues,
			IgnoreCacErrors:                 args.IgnoreCacErrors,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client:                &octopusClient,
			LimitResourceCount:    args.LimitResourceCount,
			GenerateImportScripts: args.GenerateImportScripts,
			EnvironmentConverter:  environmentConverter,
			IncludeIds:            args.IncludeIds,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		RunbookConverter:          &runbookConverter,
		IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
		ExcludeCaCProjectSettings: args.ExcludeCaCProjectSettings,
		ExcludeAllRunbooks:        args.ExcludeAllRunbooks,
		IgnoreProjectChanges:      args.IgnoreProjectChanges,
		IgnoreProjectGroupChanges: false,
		IgnoreProjectNameChanges:  false,
		ExcludeProjects:           args.ExcludeProjects,
		ExcludeProjectsExcept:     args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:      args.ExcludeProjectsRegex,
		ExcludeAllProjects:        args.ExcludeAllProjects,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		Excluder:                  converters.DefaultExcluder{},
		LookupOnlyMode:            false,
		ErrGroup:                  &group,
		ExcludeTerraformVariables: args.ExcludeTerraformVariables,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		LookupProjectLinkTenants:  false,
		TenantProjectConverter:    tenantProjectConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeTenantsRegex:       args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:    args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		IgnoreCacErrors:           args.IgnoreCacErrors,
	}

	deploymentFreezeConverter := converters.DeploymentFreezeConverter{
		Client:                         &octopusClient,
		ErrGroup:                       &group,
		ExcludeDeploymentFreezes:       args.ExcludeDeploymentFreezes,
		ExcludeDeploymentFreezesRegex:  args.ExcludeDeploymentFreezesRegex,
		ExcludeDeploymentFreezesExcept: args.ExcludeDeploymentFreezesExcept,
		ExcludeAllDeploymentFreezes:    args.ExcludeAllDeploymentFreezes,
		Excluder:                       converters.DefaultExcluder{},
		LimitResourceCount:             args.LimitResourceCount,
		IncludeIds:                     args.IncludeIds,
		GenerateImportScripts:          args.GenerateImportScripts,
	}

	spaceConverter := converters.SpaceConverter{
		Client:                      &octopusClient,
		ExcludeSpaceCreation:        args.ExcludeSpaceCreation,
		AccountConverter:            accountConverter,
		EnvironmentConverter:        environmentConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		LifecycleConverter:          lifecycleConverter,
		WorkerPoolConverter:         workerPoolConverter,
		TagSetConverter:             &tagsetConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		ProjectGroupConverter:       projectGroupConverter,
		SpacePopulateConverter: converters.SpacePopulateConverter{
			Client:                   &octopusClient,
			IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
			IncludeIds:               args.IncludeIds,
			ErrGroup:                 &group,
		},
		ProjectConverter:                  projectConverter,
		TenantConverter:                   &tenantConverter,
		CertificateConverter:              certificateConverter,
		TenantVariableConverter:           tenantVariableConverter,
		MachinePolicyConverter:            machinePolicyConverter,
		KubernetesTargetConverter:         kubernetesTargetConverter,
		SshTargetConverter:                sshTargetConverter,
		ListeningTargetConverter:          listeningTargetConverter,
		PollingTargetConverter:            pollingTargetConverter,
		CloudRegionTargetConverter:        cloudRegionTargetConverter,
		OfflineDropTargetConverter:        offlineDropTargetConverter,
		AzureCloudServiceTargetConverter:  azureCloudServiceTargetConverter,
		AzureServiceFabricTargetConverter: azureServiceFabricTargetConverter,
		AzureWebAppTargetConverter:        azureWebAppTargetConverter,
		FeedConverter:                     feedConverter,
		ErrGroup:                          &group,
		StepTemplateConverter:             stepTemplateConverter,
		TenantProjectConverter:            tenantProjectConverter,
		DeploymentFreezeConverter:         deploymentFreezeConverter,
		KubernetesAgentWorkerConverter:    k8sAgentWorkerConverter,
		ListeningWorkerConverter:          listeningWorkerConverter,
	}

	octopusActionProcessor := converters.OctopusActionProcessor{
		FeedConverter:                   feedConverter,
		AccountConverter:                accountConverter,
		WorkerPoolConverter:             workerPoolConverter,
		EnvironmentConverter:            environmentConverter,
		DetachProjectTemplates:          args.DetachProjectTemplates,
		WorkerPoolProcessor:             workerPoolProcessor,
		GitCredentialsConverter:         gitCredentialsConverter,
		StepTemplateConverter:           stepTemplateConverter,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
		ProjectExporter:                 projectConverter,
	}

	// Projects and runbooks have circular references to other projects. For example, a project can have
	// a "Deploy a release" step, which references another project.
	// The ProjectExporter field on the OctopusActionProcessor is set to the same instance of the project converter
	// that references the OctopusActionProcessor.
	runbookConverter.RunbookProcessConverter.SetActionProcessor(&octopusActionProcessor)
	projectConverter.DeploymentProcessConverter.SetActionProcessor(&octopusActionProcessor)

	if args.Stateless {
		err := spaceConverter.AllToStatelessHcl(&dependencies)

		if err != nil {
			return nil, err
		}
	} else {
		err := spaceConverter.AllToHcl(&dependencies)

		if err != nil {
			return nil, err
		}
	}

	return &dependencies, nil
}

func ConvertRunbookToTerraform(args args.Arguments, version string) (*data.ResourceDetailsCollection, error) {

	octopusClient := client.OctopusApiClient{
		Url:     args.Url,
		Space:   args.Space,
		ApiKey:  args.ApiKey,
		Version: version,
	}

	dummySecretGenerator := dummy.DummySecret{}

	tenantCommonVariableProcessor := converters.TenantCommonVariableProcessor{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
	}

	tenantProjectVariableConverter := converters.TenantProjectVariableConverter{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
		DummySecretVariableValues:    args.DummySecretVariableValues,
		DummySecretGenerator:         dummySecretGenerator,
	}

	tenantProjectConverter := converters.TenantProjectConverter{
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		ErrGroup:                 nil,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		Excluder:                 converters.DefaultExcluder{},
		Client:                   &octopusClient,
	}

	dependencies := data.ResourceDetailsCollection{}

	stepTemplateConverter := converters.StepTemplateConverter{
		ErrGroup:                        nil,
		Client:                          &octopusClient,
		ExcludeAllStepTemplates:         false,
		ExcludeStepTemplates:            nil,
		ExcludeStepTemplatesRegex:       nil,
		ExcludeStepTemplatesExcept:      nil,
		Excluder:                        converters.DefaultExcluder{},
		LimitResourceCount:              0,
		GenerateImportScripts:           false,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
	}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	environmentConverter := converters.EnvironmentConverter{
		Client:                    &octopusClient,
		ExcludeEnvironments:       args.ExcludeEnvironments,
		ExcludeAllEnvironments:    args.ExcludeAllEnvironments,
		ExcludeEnvironmentsExcept: args.ExcludeEnvironmentsExcept,
		ExcludeEnvironmentsRegex:  args.ExcludeEnvironmentsRegex,
		Excluder:                  converters.DefaultExcluder{},
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
		ErrGroup:                  nil,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:                     &octopusClient,
		Excluder:                   converters.DefaultExcluder{},
		ExcludeTenantTags:          args.ExcludeTenantTags,
		ExcludeTenantTagSets:       args.ExcludeTenantTagSets,
		ExcludeTenantTagSetsRegex:  args.ExcludeTenantTagSetsRegex,
		ExcludeTenantTagSetsExcept: args.ExcludeTenantTagSetsExcept,
		ExcludeAllTenantTagSets:    args.ExcludeAllTenantTagSets,
		ErrGroup:                   nil,
		LimitResourceCount:         args.LimitResourceCount,
		GenerateImportScripts:      args.GenerateImportScripts,
	}

	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                         &octopusClient,
		ExcludeTenants:                 args.ExcludeTenants,
		ExcludeTenantsWithTags:         nil,
		ExcludeTenantsExcept:           args.ExcludeTenantsExcept,
		ExcludeAllTenants:              args.ExcludeAllTenants,
		Excluder:                       converters.DefaultExcluder{},
		DummySecretVariableValues:      args.DummySecretVariableValues,
		DummySecretGenerator:           dummySecretGenerator,
		ExcludeProjects:                args.ExcludeProjects,
		ExcludeProjectsExcept:          args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:           args.ExcludeProjectsRegex,
		ExcludeAllProjects:             args.ExcludeAllProjects,
		ErrGroup:                       nil,
		ExcludeAllTenantVariables:      args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:         args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept:   args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:    args.ExcludeTenantVariablesRegex,
		TenantCommonVariableProcessor:  tenantCommonVariableProcessor,
		TenantProjectVariableConverter: tenantProjectVariableConverter,
	}
	tenantConverter := converters.TenantConverter{
		Client:                   &octopusClient,
		TenantVariableConverter:  tenantVariableConverter,
		EnvironmentConverter:     environmentConverter,
		TagSetConverter:          &tagsetConverter,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		Excluder:                 converters.DefaultExcluder{},
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		ErrGroup:                 nil,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		TenantProjectConverter:   tenantProjectConverter,
	}

	accountConverter := converters.AccountConverter{
		Client:                    &octopusClient,
		EnvironmentConverter:      environmentConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  nil,
		ExcludeAccounts:           args.ExcludeAccounts,
		ExcludeAccountsRegex:      args.ExcludeAccountsRegex,
		ExcludeAccountsExcept:     args.ExcludeAccountsExcept,
		ExcludeAllAccounts:        args.ExcludeAllAccounts,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	feedConverter := converters.FeedConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeFeeds:              args.ExcludeFeeds,
		ExcludeFeedsRegex:         args.ExcludeFeedsRegex,
		ExcludeFeedsExcept:        args.ExcludeFeedsExcept,
		ExcludeAllFeeds:           args.ExcludeAllFeeds,
		Excluder:                  converters.DefaultExcluder{},
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}
	workerPoolConverter := converters.WorkerPoolConverter{
		Client:                   &octopusClient,
		ErrGroup:                 nil,
		ExcludeWorkerpools:       args.ExcludeWorkerpools,
		ExcludeWorkerpoolsRegex:  args.ExcludeWorkerpoolsRegex,
		ExcludeWorkerpoolsExcept: args.ExcludeWorkerpoolsExcept,
		ExcludeAllWorkerpools:    args.ExcludeAllWorkerpools,
		Excluder:                 converters.DefaultExcluder{},
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  &octopusClient,
	}

	projectConverter := &converters.ProjectConverter{
		Client:                      &octopusClient,
		LifecycleConverter:          nil,
		GitCredentialsConverter:     nil,
		LibraryVariableSetConverter: nil,
		ProjectGroupConverter:       nil,
		DeploymentProcessConverter:  nil,
		TenantConverter:             nil,
		ProjectTriggerConverter:     nil,
		VariableSetConverter:        nil,
		ChannelConverter:            nil,
		RunbookConverter:            nil,
		IgnoreCacManagedValues:      false,
		ExcludeCaCProjectSettings:   false,
		ExcludeAllRunbooks:          false,
		IgnoreProjectChanges:        false,
		IgnoreProjectGroupChanges:   false,
		IgnoreProjectNameChanges:    false,
		ExcludeProjects:             nil,
		ExcludeProjectsExcept:       nil,
		ExcludeProjectsRegex:        nil,
		ExcludeAllProjects:          false,
		DummySecretVariableValues:   false,
		DummySecretGenerator:        nil,
		Excluder:                    converters.DefaultExcluder{},
		LookupOnlyMode:              true,
		ErrGroup:                    nil,
		ExcludeTerraformVariables:   args.ExcludeTerraformVariables,
		IncludeIds:                  args.IncludeIds,
		LimitResourceCount:          args.LimitResourceCount,
		IncludeSpaceInPopulation:    args.IncludeSpaceInPopulation,
		GenerateImportScripts:       args.GenerateImportScripts,
		TenantVariableConverter:     tenantVariableConverter,
		LookupProjectLinkTenants:    false,
		TenantProjectConverter:      tenantProjectConverter,
		EnvironmentConverter:        environmentConverter,
		ExcludeTenantTagSets:        args.ExcludeTenantTagSets,
		ExcludeTenantTags:           args.ExcludeTenantTags,
		ExcludeTenants:              args.ExcludeTenants,
		ExcludeTenantsRegex:         args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:      args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:        args.ExcludeTenantsExcept,
		ExcludeAllTenants:           args.ExcludeAllTenants,
		IgnoreCacErrors:             args.IgnoreCacErrors,
	}

	runbookConverter := converters.RunbookConverter{
		Client: &octopusClient,
		RunbookProcessConverter: &converters.RunbookProcessConverter{
			Client:                          &octopusClient,
			OctopusActionProcessor:          nil,
			IgnoreProjectChanges:            args.IgnoreProjectChanges,
			WorkerPoolProcessor:             workerPoolProcessor,
			ExcludeTenantTags:               args.ExcludeTenantTags,
			ExcludeTenantTagSets:            args.ExcludeTenantTagSets,
			Excluder:                        converters.DefaultExcluder{},
			TagSetConverter:                 &tagsetConverter,
			LimitAttributeLength:            0,
			ExcludeAllSteps:                 args.ExcludeAllSteps,
			ExcludeSteps:                    args.ExcludeSteps,
			ExcludeStepsRegex:               args.ExcludeStepsRegex,
			ExcludeStepsExcept:              args.ExcludeStepsExcept,
			IgnoreInvalidExcludeExcept:      args.IgnoreInvalidExcludeExcept,
			ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
			DummySecretGenerator:            dummySecretGenerator,
			DummySecretVariableValues:       args.DummySecretVariableValues,
		},
		EnvironmentConverter:     environmentConverter,
		ExcludedRunbooks:         nil,
		ExcludeRunbooksRegex:     nil,
		ExcludeRunbooksExcept:    nil,
		ExcludeAllRunbooks:       false,
		Excluder:                 converters.DefaultExcluder{},
		IgnoreProjectChanges:     args.IgnoreProjectChanges,
		ProjectConverter:         projectConverter,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	octopusActionProcessor := converters.OctopusActionProcessor{
		FeedConverter:                   feedConverter,
		AccountConverter:                accountConverter,
		WorkerPoolConverter:             workerPoolConverter,
		EnvironmentConverter:            environmentConverter,
		DetachProjectTemplates:          args.DetachProjectTemplates,
		WorkerPoolProcessor:             workerPoolProcessor,
		GitCredentialsConverter:         gitCredentialsConverter,
		StepTemplateConverter:           stepTemplateConverter,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
		ProjectExporter:                 projectConverter,
	}

	runbookConverter.RunbookProcessConverter.SetActionProcessor(&octopusActionProcessor)

	err := runbookConverter.ToHclByIdWithLookups(args.RunbookId, &dependencies)

	if err != nil {
		return nil, err
	}

	return &dependencies, nil
}

func ConvertProjectToTerraform(args args.Arguments, version string) (*data.ResourceDetailsCollection, error) {

	octopusClient := client.OctopusApiClient{
		Url:     args.Url,
		Space:   args.Space,
		ApiKey:  args.ApiKey,
		Version: version,
	}

	dummySecretGenerator := dummy.DummySecret{}

	tenantCommonVariableProcessor := converters.TenantCommonVariableProcessor{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
	}

	tenantProjectVariableConverter := converters.TenantProjectVariableConverter{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           args.ExcludeAllProjects,
		ExcludeAllTenantVariables:    args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:       args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept: args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:  args.ExcludeTenantVariablesRegex,
		DummySecretVariableValues:    args.DummySecretVariableValues,
		DummySecretGenerator:         dummySecretGenerator,
	}

	tenantProjectConverter := converters.TenantProjectConverter{
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		ErrGroup:                 nil,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		Excluder:                 converters.DefaultExcluder{},
		Client:                   &octopusClient,
	}

	dependencies := data.ResourceDetailsCollection{}

	stepTemplateConverter := converters.StepTemplateConverter{
		ErrGroup:                        nil,
		Client:                          &octopusClient,
		ExcludeAllStepTemplates:         false,
		ExcludeStepTemplates:            nil,
		ExcludeStepTemplatesRegex:       nil,
		ExcludeStepTemplatesExcept:      nil,
		Excluder:                        converters.DefaultExcluder{},
		LimitResourceCount:              0,
		GenerateImportScripts:           false,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
	}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	environmentConverter := converters.EnvironmentConverter{
		Client:                    &octopusClient,
		ExcludeEnvironments:       args.ExcludeEnvironments,
		ExcludeAllEnvironments:    args.ExcludeAllEnvironments,
		ExcludeEnvironmentsExcept: args.ExcludeEnvironmentsExcept,
		ExcludeEnvironmentsRegex:  args.ExcludeEnvironmentsRegex,
		Excluder:                  converters.DefaultExcluder{},
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}
	lifecycleConverter := converters.LifecycleConverter{
		Client:                   &octopusClient,
		EnvironmentConverter:     environmentConverter,
		ErrGroup:                 nil,
		ExcludeLifecycles:        args.ExcludeLifecycles,
		ExcludeLifecyclesRegex:   args.ExcludeLifecyclesRegex,
		ExcludeLifecyclesExcept:  args.ExcludeLifecyclesExcept,
		ExcludeAllLifecycles:     args.ExcludeAllLifecycles,
		Excluder:                 converters.DefaultExcluder{},
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
		ErrGroup:                  nil,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:                     &octopusClient,
		ExcludeTenantTags:          args.ExcludeTenantTags,
		ExcludeTenantTagSets:       args.ExcludeTenantTagSets,
		ExcludeTenantTagSetsRegex:  args.ExcludeTenantTagSetsRegex,
		ExcludeTenantTagSetsExcept: args.ExcludeTenantTagSetsExcept,
		ExcludeAllTenantTagSets:    args.ExcludeAllTenantTagSets,
		Excluder:                   converters.DefaultExcluder{},
		ErrGroup:                   nil,
		LimitResourceCount:         args.LimitResourceCount,
		GenerateImportScripts:      args.GenerateImportScripts,
	}
	channelConverter := converters.ChannelConverter{
		Client:                   &octopusClient,
		LifecycleConverter:       lifecycleConverter,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		Excluder:                 converters.DefaultExcluder{},
		ErrGroup:                 nil,
		IncludeIds:               false,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeDefaultChannel:    args.IncludeDefaultChannel,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IgnoreCacErrors:          args.IgnoreCacErrors,
		ExcludeAllChannels:       args.ExcludeAllChannels,
		ExcludeChannels:          args.ExcludeChannels,
		ExcludeChannelsRegex:     args.ExcludeChannelsRegex,
		ExcludeChannelsExcept:    args.ExcludeChannelsExcept,
		ExcludeInvalidChannels:   args.ExcludeInvalidChannels,
	}

	projectGroupConverter := converters.ProjectGroupConverter{
		Client:                     &octopusClient,
		ErrGroup:                   nil,
		ExcludeProjectGroups:       args.ExcludeProjectGroups,
		ExcludeProjectGroupsRegex:  args.ExcludeProjectGroupsRegex,
		ExcludeProjectGroupsExcept: args.ExcludeProjectGroupsExcept,
		ExcludeAllProjectGroups:    args.ExcludeAllProjectGroups,
		Excluder:                   converters.DefaultExcluder{},
		LimitResourceCount:         args.LimitResourceCount,
		IncludeIds:                 args.IncludeIds,
		IncludeSpaceInPopulation:   args.IncludeSpaceInPopulation,
		GenerateImportScripts:      args.GenerateImportScripts,
	}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                         &octopusClient,
		ExcludeTenants:                 args.ExcludeTenants,
		ExcludeTenantsWithTags:         args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:           args.ExcludeTenantsExcept,
		ExcludeAllTenants:              args.ExcludeAllTenants,
		Excluder:                       converters.DefaultExcluder{},
		DummySecretVariableValues:      args.DummySecretVariableValues,
		DummySecretGenerator:           dummySecretGenerator,
		ExcludeProjects:                args.ExcludeProjects,
		ExcludeProjectsExcept:          args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:           args.ExcludeProjectsRegex,
		ExcludeAllProjects:             args.ExcludeAllProjects,
		ErrGroup:                       nil,
		ExcludeAllTenantVariables:      args.ExcludeAllTenantVariables,
		ExcludeTenantVariables:         args.ExcludeTenantVariables,
		ExcludeTenantVariablesExcept:   args.ExcludeTenantVariablesExcept,
		ExcludeTenantVariablesRegex:    args.ExcludeTenantVariablesRegex,
		TenantCommonVariableProcessor:  tenantCommonVariableProcessor,
		TenantProjectVariableConverter: tenantProjectVariableConverter,
	}
	tenantConverter := converters.TenantConverter{
		Client:                   &octopusClient,
		TenantVariableConverter:  tenantVariableConverter,
		EnvironmentConverter:     environmentConverter,
		TagSetConverter:          &tagsetConverter,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenants:           args.ExcludeTenants,
		ExcludeTenantsRegex:      args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:   args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:     args.ExcludeTenantsExcept,
		ExcludeAllTenants:        args.ExcludeAllTenants,
		Excluder:                 converters.DefaultExcluder{},
		ExcludeProjects:          args.ExcludeProjects,
		ExcludeProjectsExcept:    args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:     args.ExcludeProjectsRegex,
		ExcludeAllProjects:       args.ExcludeAllProjects,
		ErrGroup:                 nil,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		TenantProjectConverter:   tenantProjectConverter,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{
		Client:                       &octopusClient,
		ExcludeMachinePolicies:       args.ExcludeMachinePolicies,
		ExcludeMachinePoliciesRegex:  args.ExcludeMachinePoliciesRegex,
		ExcludeMachinePoliciesExcept: args.ExcludeMachinePoliciesExcept,
		ExcludeAllMachinePolicies:    args.ExcludeAllMachinePolicies,
		Excluder:                     converters.DefaultExcluder{},
		LimitResourceCount:           args.LimitResourceCount,
		IncludeIds:                   args.IncludeIds,
		IncludeSpaceInPopulation:     args.IncludeSpaceInPopulation,
		GenerateImportScripts:        args.GenerateImportScripts,
		ErrGroup:                     nil,
	}
	accountConverter := converters.AccountConverter{
		Client:                    &octopusClient,
		EnvironmentConverter:      environmentConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ExcludeAccounts:           args.ExcludeAccounts,
		ExcludeAccountsRegex:      args.ExcludeAccountsRegex,
		ExcludeAccountsExcept:     args.ExcludeAccountsExcept,
		ExcludeAllAccounts:        args.ExcludeAllAccounts,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}
	certificateConverter := converters.CertificateConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  nil,
		ExcludeCertificates:       args.ExcludeCertificates,
		ExcludeCertificatesRegex:  args.ExcludeCertificatesRegex,
		ExcludeCertificatesExcept: args.ExcludeCertificatesExcept,
		ExcludeAllCertificates:    args.ExcludeAllCertificates,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeIds:                args.IncludeIds,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
	}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		CertificateConverter:     certificateConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeIds:               args.IncludeIds,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	sshTargetConverter := converters.SshTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ErrGroup:                 nil,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &octopusClient,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              args.ExcludeEnvironments,
			ExcludeEnvironmentsRegex:         args.ExcludeEnvironmentsRegex,
			ExcludeEnvironmentsExcept:        args.ExcludeEnvironmentsExcept,
			ExcludeAllEnvironments:           args.ExcludeAllEnvironments,
			ExcludeTargetsWithNoEnvironments: args.ExcludeTargetsWithNoEnvironments,
		},
		MachinePolicyConverter:   machinePolicyConverter,
		AccountConverter:         accountConverter,
		EnvironmentConverter:     environmentConverter,
		ExcludeAllTargets:        args.ExcludeAllTargets,
		ExcludeTenantTags:        args.ExcludeTenantTags,
		ExcludeTenantTagSets:     args.ExcludeTenantTagSets,
		TagSetConverter:          &tagsetConverter,
		ExcludeTargets:           args.ExcludeTargets,
		ExcludeTargetsRegex:      args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:     args.ExcludeTargetsExcept,
		IncludeIds:               args.IncludeIds,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	feedConverter := converters.FeedConverter{
		Client:                    &octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeFeeds:              args.ExcludeFeeds,
		ExcludeFeedsRegex:         args.ExcludeFeedsRegex,
		ExcludeFeedsExcept:        args.ExcludeFeedsExcept,
		ExcludeAllFeeds:           args.ExcludeAllFeeds,
		Excluder:                  converters.DefaultExcluder{},
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		ErrGroup:                  nil,
	}
	workerPoolConverter := converters.WorkerPoolConverter{
		Client:                   &octopusClient,
		ErrGroup:                 nil,
		ExcludeWorkerpools:       args.ExcludeWorkerpools,
		ExcludeWorkerpoolsRegex:  args.ExcludeWorkerpoolsRegex,
		ExcludeWorkerpoolsExcept: args.ExcludeWorkerpoolsExcept,
		ExcludeAllWorkerpools:    args.ExcludeAllWorkerpools,
		Excluder:                 converters.DefaultExcluder{},
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            &octopusClient,
		ChannelConverter:                  channelConverter,
		EnvironmentConverter:              environmentConverter,
		TagSetConverter:                   &tagsetConverter,
		AzureCloudServiceTargetConverter:  azureCloudServiceTargetConverter,
		AzureServiceFabricTargetConverter: azureServiceFabricTargetConverter,
		AzureWebAppTargetConverter:        azureWebAppTargetConverter,
		CloudRegionTargetConverter:        cloudRegionTargetConverter,
		KubernetesTargetConverter:         kubernetesTargetConverter,
		ListeningTargetConverter:          listeningTargetConverter,
		OfflineDropTargetConverter:        offlineDropTargetConverter,
		PollingTargetConverter:            pollingTargetConverter,
		SshTargetConverter:                sshTargetConverter,
		AccountConverter:                  accountConverter,
		FeedConverter:                     feedConverter,
		CertificateConverter:              certificateConverter,
		WorkerPoolConverter:               workerPoolConverter,
		IgnoreCacManagedValues:            args.IgnoreCacManagedValues,
		DefaultSecretVariableValues:       args.DefaultSecretVariableValues,
		DummySecretVariableValues:         args.DummySecretVariableValues,
		ExcludeAllProjectVariables:        args.ExcludeAllProjectVariables,
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesExcept:     args.ExcludeProjectVariablesExcept,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeTenantTagSets:              args.ExcludeTenantTagSets,
		ExcludeTenantTags:                 args.ExcludeTenantTags,
		IgnoreProjectChanges:              args.IgnoreProjectChanges || args.IgnoreProjectVariableChanges,
		DummySecretGenerator:              dummySecretGenerator,
		Excluder:                          converters.DefaultExcluder{},
		ErrGroup:                          nil,
		ExcludeTerraformVariables:         args.ExcludeTerraformVariables,
		LimitAttributeLength:              args.LimitAttributeLength,
		StatelessAdditionalParams:         args.StatelessAdditionalParams,
		GenerateImportScripts:             args.GenerateImportScripts,
		EnvironmentFilter: converters.EnvironmentFilter{
			Client:                           &octopusClient,
			ExcludeVariableEnvironmentScopes: args.ExcludeVariableEnvironmentScopes,
		},
		IgnoreCacErrors: args.IgnoreCacErrors,
	}

	variableSetConverterForLibrary := converters.VariableSetConverter{
		Client:                            &octopusClient,
		ChannelConverter:                  channelConverter,
		EnvironmentConverter:              environmentConverter,
		TagSetConverter:                   &tagsetConverter,
		AzureCloudServiceTargetConverter:  azureCloudServiceTargetConverter,
		AzureServiceFabricTargetConverter: azureServiceFabricTargetConverter,
		AzureWebAppTargetConverter:        azureWebAppTargetConverter,
		CloudRegionTargetConverter:        cloudRegionTargetConverter,
		KubernetesTargetConverter:         kubernetesTargetConverter,
		ListeningTargetConverter:          listeningTargetConverter,
		OfflineDropTargetConverter:        offlineDropTargetConverter,
		PollingTargetConverter:            pollingTargetConverter,
		SshTargetConverter:                sshTargetConverter,
		AccountConverter:                  accountConverter,
		FeedConverter:                     feedConverter,
		CertificateConverter:              certificateConverter,
		WorkerPoolConverter:               workerPoolConverter,
		IgnoreCacManagedValues:            args.IgnoreCacManagedValues,
		DefaultSecretVariableValues:       args.DefaultSecretVariableValues,
		DummySecretVariableValues:         args.DummySecretVariableValues,
		ExcludeAllProjectVariables:        args.ExcludeAllProjectVariables,
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesExcept:     args.ExcludeProjectVariablesExcept,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeTenantTagSets:              args.ExcludeTenantTagSets,
		ExcludeTenantTags:                 args.ExcludeTenantTags,
		IgnoreProjectChanges:              args.IgnoreProjectChanges,
		DummySecretGenerator:              dummySecretGenerator,
		Excluder:                          converters.DefaultExcluder{},
		ErrGroup:                          nil,
		ExcludeTerraformVariables:         args.ExcludeTerraformVariables,
		LimitAttributeLength:              args.LimitAttributeLength,
		StatelessAdditionalParams:         args.StatelessAdditionalParams,
		GenerateImportScripts:             args.GenerateImportScripts,
		EnvironmentFilter: converters.EnvironmentFilter{
			Client:                           &octopusClient,
			ExcludeVariableEnvironmentScopes: args.ExcludeVariableEnvironmentScopes,
		},
		IgnoreCacErrors: args.IgnoreCacErrors,
	}

	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                           &octopusClient,
		VariableSetConverter:             &variableSetConverterForLibrary,
		Excluded:                         args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex:  args.ExcludeLibraryVariableSetsRegex,
		ExcludeLibraryVariableSetsExcept: args.ExcludeLibraryVariableSetsExcept,
		ExcludeAllLibraryVariableSets:    args.ExcludeAllLibraryVariableSets,
		DummySecretVariableValues:        args.DummySecretVariableValues,
		DummySecretGenerator:             dummySecretGenerator,
		Excluder:                         converters.DefaultExcluder{},
		LimitResourceCount:               args.LimitResourceCount,
		GenerateImportScripts:            args.GenerateImportScripts,
		ErrGroup:                         nil,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  &octopusClient,
		ErrGroup:                nil,
	}

	runbookConverter := converters.RunbookConverter{
		Client: &octopusClient,
		RunbookProcessConverter: &converters.RunbookProcessConverter{
			Client:                          &octopusClient,
			OctopusActionProcessor:          nil,
			IgnoreProjectChanges:            args.IgnoreProjectChanges,
			WorkerPoolProcessor:             workerPoolProcessor,
			ExcludeTenantTags:               args.ExcludeTenantTags,
			ExcludeTenantTagSets:            args.ExcludeTenantTagSets,
			Excluder:                        converters.DefaultExcluder{},
			TagSetConverter:                 &tagsetConverter,
			LimitAttributeLength:            0,
			ExcludeAllSteps:                 args.ExcludeAllSteps,
			ExcludeSteps:                    args.ExcludeSteps,
			ExcludeStepsRegex:               args.ExcludeStepsRegex,
			ExcludeStepsExcept:              args.ExcludeStepsExcept,
			IgnoreInvalidExcludeExcept:      args.IgnoreInvalidExcludeExcept,
			ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
			DummySecretGenerator:            dummySecretGenerator,
			DummySecretVariableValues:       args.DummySecretVariableValues,
		},
		EnvironmentConverter:     environmentConverter,
		ProjectConverter:         nil,
		ExcludedRunbooks:         args.ExcludeRunbooks,
		ExcludeRunbooksRegex:     args.ExcludeRunbooksRegex,
		ExcludeRunbooksExcept:    args.ExcludeRunbooksExcept,
		ExcludeAllRunbooks:       args.ExcludeAllRunbooks,
		Excluder:                 converters.DefaultExcluder{},
		IgnoreProjectChanges:     args.IgnoreProjectChanges,
		LimitResourceCount:       args.LimitResourceCount,
		IncludeSpaceInPopulation: args.IncludeSpaceInPopulation,
		IncludeIds:               args.IncludeIds,
		GenerateImportScripts:    args.GenerateImportScripts,
		ErrGroup:                 nil,
	}

	projectConverter := converters.ProjectConverter{
		Client:                      &octopusClient,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: &converters.DeploymentProcessConverter{
			Client:                          &octopusClient,
			OctopusActionProcessor:          nil,
			IgnoreProjectChanges:            args.IgnoreProjectChanges,
			WorkerPoolProcessor:             workerPoolProcessor,
			ExcludeTenantTags:               args.ExcludeTenantTags,
			ExcludeTenantTagSets:            args.ExcludeTenantTagSets,
			Excluder:                        converters.DefaultExcluder{},
			TagSetConverter:                 &tagsetConverter,
			LimitAttributeLength:            0,
			ExcludeTerraformVariables:       args.ExcludeTerraformVariables,
			ExcludeAllSteps:                 args.ExcludeAllSteps,
			ExcludeSteps:                    args.ExcludeSteps,
			ExcludeStepsRegex:               args.ExcludeStepsRegex,
			ExcludeStepsExcept:              args.ExcludeStepsExcept,
			IgnoreInvalidExcludeExcept:      args.IgnoreInvalidExcludeExcept,
			ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
			DummySecretGenerator:            dummySecretGenerator,
			DummySecretVariableValues:       args.DummySecretVariableValues,
			IgnoreCacErrors:                 args.IgnoreCacErrors,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client:                &octopusClient,
			LimitResourceCount:    args.LimitResourceCount,
			GenerateImportScripts: args.GenerateImportScripts,
			EnvironmentConverter:  environmentConverter,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		RunbookConverter:          &runbookConverter,
		EnvironmentConverter:      environmentConverter,
		IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
		ExcludeCaCProjectSettings: args.ExcludeCaCProjectSettings,
		ExcludeAllRunbooks:        args.ExcludeAllRunbooks,
		IgnoreProjectChanges:      args.IgnoreProjectChanges,
		IgnoreProjectGroupChanges: args.IgnoreProjectGroupChanges,
		IgnoreProjectNameChanges:  args.IgnoreProjectNameChanges,
		ExcludeProjects:           nil,
		ExcludeProjectsExcept:     nil,
		ExcludeProjectsRegex:      nil,
		ExcludeAllProjects:        false,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		Excluder:                  converters.DefaultExcluder{},
		LookupOnlyMode:            false,
		ErrGroup:                  nil,
		ExcludeTerraformVariables: args.ExcludeTerraformVariables,
		IncludeIds:                args.IncludeIds,
		LimitResourceCount:        args.LimitResourceCount,
		IncludeSpaceInPopulation:  args.IncludeSpaceInPopulation,
		GenerateImportScripts:     args.GenerateImportScripts,
		LookupProjectLinkTenants:  args.LookupProjectLinkTenants,
		TenantProjectConverter:    tenantProjectConverter,
		TenantVariableConverter:   tenantVariableConverter,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeTenantsRegex:       args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:    args.ExcludeTenantsWithTags,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		IgnoreCacErrors:           args.IgnoreCacErrors,
	}

	octopusActionProcessor := converters.OctopusActionProcessor{
		FeedConverter:                   feedConverter,
		AccountConverter:                accountConverter,
		WorkerPoolConverter:             workerPoolConverter,
		EnvironmentConverter:            environmentConverter,
		DetachProjectTemplates:          args.DetachProjectTemplates,
		WorkerPoolProcessor:             workerPoolProcessor,
		GitCredentialsConverter:         gitCredentialsConverter,
		StepTemplateConverter:           stepTemplateConverter,
		ExperimentalEnableStepTemplates: args.ExperimentalEnableStepTemplates,
		ProjectExporter:                 &projectConverter,
	}

	// Projects and runbooks have circular references to other projects. For example, a project can have
	// a "Deploy a release" step, which references another project.
	// The ProjectExporter field on the OctopusActionProcessor is set to the same instance of the project converter
	// that references the OctopusActionProcessor.
	projectConverter.DeploymentProcessConverter.SetActionProcessor(&octopusActionProcessor)
	runbookConverter.RunbookProcessConverter.SetActionProcessor(&octopusActionProcessor)

	if args.LookupProjectDependencies {
		for _, project := range args.ProjectId {
			err := projectConverter.ToHclByIdWithLookups(project, &dependencies)

			if err != nil {
				return nil, err
			}
		}
	} else {
		if args.Stateless {
			for _, project := range args.ProjectId {
				err := projectConverter.ToHclStatelessById(project, &dependencies)

				if err != nil {
					return nil, err
				}
			}
		} else {
			for _, project := range args.ProjectId {
				err := projectConverter.ToHclById(project, &dependencies)

				if err != nil {
					return nil, err
				}
			}
		}
	}

	return &dependencies, nil
}

// ProcessResources creates a map of file names to file content
func ProcessResources(resources []data.ResourceDetails) (map[string]string, error) {
	zap.L().Info("Generating HCL (this can take a little while)")
	defer zap.L().Info("Done Generating HCL")

	var wg sync.WaitGroup
	var fileMap sync.Map
	hclErrors := collections.SafeErrorSlice{}

	for _, r := range resources {
		// Some resources are already resolved by their parent, but exist in the resource details map as a lookup.
		// In these cases, ToHclByProjectId is nil.
		if r.ToHcl == nil {
			continue
		}

		wg.Add(1)

		resource := r
		go func() {
			defer wg.Done()
			hcl, err := resource.ToHcl()

			if err != nil {
				hclErrors.Append(err)
			} else {
				if len(strings.TrimSpace(hcl)) != 0 {
					fileMap.Store(resource.FileName, hcl)
				}
			}
		}()
	}

	wg.Wait()
	if len(hclErrors.GetCopy()) != 0 {
		return nil, errors.Join(hclErrors.GetCopy()...)
	}

	result := map[string]string{}
	fileMap.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(string)
		return true
	})

	return result, nil
}
