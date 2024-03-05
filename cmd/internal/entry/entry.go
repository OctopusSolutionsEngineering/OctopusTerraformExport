package entry

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/collections"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/converters"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/generators"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
)

// Entry takes the arguments, exports the Octopus resources to HCL in strings and returns the strings mapped to file names.
func Entry(parseArgs args.Arguments) (map[string]string, error) {

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
			projectId, err := ConvertProjectNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, project)

			if err != nil {
				return nil, err
			}

			projectIds = append(projectIds, projectId)
		}

		parseArgs.ProjectId = projectIds
	}

	if parseArgs.RunbookName != "" {
		runbookId, err := ConvertRunbookNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, parseArgs.ProjectId[0], parseArgs.RunbookName)

		if err != nil {
			return nil, err
		}

		parseArgs.RunbookId = runbookId
	}

	if parseArgs.RunbookId != "" {
		zap.L().Info("Exporting runbook " + parseArgs.RunbookId + " in space " + parseArgs.Space)
		files, err := ConvertRunbookToTerraform(parseArgs)
		if err != nil {
			return nil, err
		}
		return files, nil
	} else if len(parseArgs.ProjectId) != 0 {
		zap.L().Info("Exporting project(s) " + strings.Join(parseArgs.ProjectId, ", ") + " in space " + parseArgs.Space)
		files, err := ConvertProjectToTerraform(parseArgs)
		if err != nil {
			return nil, err
		}
		return files, nil
	} else {
		zap.L().Info("Exporting space " + parseArgs.Space)
		files, err := ConvertSpaceToTerraform(parseArgs)
		if err != nil {
			return nil, err
		}
		return files, nil
	}
}

func ConvertProjectNameToId(url string, space string, apiKey string, name string) (string, error) {
	octopusClient := client.OctopusApiClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
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

func ConvertRunbookNameToId(url string, space string, apiKey string, projectId string, runbookName string) (string, error) {
	octopusClient := client.OctopusApiClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
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

func ConvertSpaceToTerraform(args args.Arguments) (map[string]string, error) {
	group := errgroup.Group{}
	group.SetLimit(10)

	octopusClient := client.OctopusApiClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dependencies := data.ResourceDetailsCollection{}

	dummySecretGenerator := converters.DummySecret{}

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
		Client:   octopusClient,
		ErrGroup: &group,
	}
	environmentConverter := converters.EnvironmentConverter{
		Client:   octopusClient,
		ErrGroup: &group,
	}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                    octopusClient,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeTenantsWithTags:    args.ExcludeTenantsWithTags,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		Excluder:                  converters.DefaultExcluder{},
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeProjects:           args.ExcludeProjects,
		ExcludeProjectsRegex:      args.ExcludeProjectsRegex,
		ExcludeAllProjects:        args.ExcludeAllProjects,
		ExcludeProjectsExcept:     args.ExcludeProjectsExcept,
		ErrGroup:                  &group,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:               octopusClient,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
		ErrGroup:             &group,
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         &tagsetConverter,
		ExcludeTenants:          args.ExcludeTenants,
		ExcludeTenantsRegex:     args.ExcludeTenantsRegex,
		ExcludeTenantsWithTags:  args.ExcludeTenantsWithTags,
		ExcludeAllTenants:       args.ExcludeAllTenants,
		ExcludeTenantsExcept:    args.ExcludeTenantsExcept,
		Excluder:                converters.DefaultExcluder{},
		ExcludeProjects:         args.ExcludeProjects,
		ExcludeProjectsRegex:    args.ExcludeProjectsRegex,
		ExcludeAllProjects:      args.ExcludeAllProjects,
		ExcludeTenantTags:       args.ExcludeTenantTags,
		ExcludeTenantTagSets:    args.ExcludeTenantTagSets,
		ExcludeProjectsExcept:   args.ExcludeProjectsExcept,
		ErrGroup:                &group,
	}
	accountConverter := converters.AccountConverter{
		Client:                    octopusClient,
		EnvironmentConverter:      machinePolicyConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  &group,
	}

	lifecycleConverter := converters.LifecycleConverter{
		Client:               octopusClient,
		EnvironmentConverter: environmentConverter,
		ErrGroup:             &group,
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
		ErrGroup:                  &group,
	}
	channelConverter := converters.ChannelConverter{
		Client:               octopusClient,
		LifecycleConverter:   lifecycleConverter,
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
		Excluder:             converters.DefaultExcluder{},
		ErrGroup:             &group,
	}

	projectGroupConverter := converters.ProjectGroupConverter{
		Client:   octopusClient,
		ErrGroup: &group,
	}

	certificateConverter := converters.CertificateConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ErrGroup:                  &group,
	}
	workerPoolConverter := converters.WorkerPoolConverter{
		Client:   octopusClient,
		ErrGroup: &group,
	}

	feedConverter := converters.FeedConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ErrGroup:                  &group,
	}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                    octopusClient,
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		ErrGroup:                  &group,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                    octopusClient,
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ErrGroup:                  &group,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ErrGroup:               &group,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            octopusClient,
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
		DefaultSecretVariableValues:       false,
		ExcludeProjectVariables:           nil,
		ExcludeProjectVariablesRegex:      nil,
		IgnoreProjectChanges:              args.IgnoreProjectChanges || args.IgnoreProjectVariableChanges,
		ExcludeVariableEnvironmentScopes:  nil,
		DummySecretVariableValues:         args.DummySecretVariableValues,
		DummySecretGenerator:              dummySecretGenerator,
		Excluder:                          converters.DefaultExcluder{},
		ErrGroup:                          &group,
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                           octopusClient,
		VariableSetConverter:             &variableSetConverter,
		Excluded:                         args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex:  args.ExcludeLibraryVariableSetsRegex,
		ExcludeLibraryVariableSetsExcept: args.ExcludeLibraryVariableSetsExcept,
		ExcludeAllLibraryVariableSets:    args.ExcludeAllLibraryVariableSets,
		DummySecretVariableValues:        args.DummySecretVariableValues,
		DummySecretGenerator:             dummySecretGenerator,
		Excluder:                         converters.DefaultExcluder{},
		ErrGroup:                         &group,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  octopusClient,
		ErrGroup:                &group,
	}

	runbookConverter := converters.RunbookConverter{
		Client: octopusClient,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: octopusClient,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:           feedConverter,
				AccountConverter:        accountConverter,
				WorkerPoolConverter:     workerPoolConverter,
				EnvironmentConverter:    environmentConverter,
				DetachProjectTemplates:  args.DetachProjectTemplates,
				WorkerPoolProcessor:     workerPoolProcessor,
				GitCredentialsConverter: gitCredentialsConverter,
			},
			IgnoreProjectChanges: false,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    args.ExcludeTenantTags,
			ExcludeTenantTagSets: args.ExcludeTenantTagSets,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		EnvironmentConverter:  environmentConverter,
		ExcludedRunbooks:      nil,
		ExcludeRunbooksRegex:  nil,
		Excluder:              converters.DefaultExcluder{},
		ExcludeRunbooksExcept: nil,
		ExcludeAllRunbooks:    false,
		ProjectConverter:      nil,
		IgnoreProjectChanges:  false,
		ErrGroup:              &group,
	}

	spaceConverter := converters.SpaceConverter{
		Client:                      octopusClient,
		AccountConverter:            accountConverter,
		EnvironmentConverter:        environmentConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		LifecycleConverter:          lifecycleConverter,
		WorkerPoolConverter:         workerPoolConverter,
		TagSetConverter:             &tagsetConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		ProjectGroupConverter:       projectGroupConverter,
		ProjectConverter: &converters.ProjectConverter{
			Client:                      octopusClient,
			LifecycleConverter:          lifecycleConverter,
			GitCredentialsConverter:     gitCredentialsConverter,
			LibraryVariableSetConverter: &libraryVariableSetConverter,
			ProjectGroupConverter:       projectGroupConverter,
			DeploymentProcessConverter: converters.DeploymentProcessConverter{
				Client: octopusClient,
				OctopusActionProcessor: converters.OctopusActionProcessor{
					FeedConverter:           feedConverter,
					AccountConverter:        accountConverter,
					WorkerPoolConverter:     workerPoolConverter,
					EnvironmentConverter:    environmentConverter,
					DetachProjectTemplates:  args.DetachProjectTemplates,
					WorkerPoolProcessor:     workerPoolProcessor,
					GitCredentialsConverter: gitCredentialsConverter,
				},
				IgnoreProjectChanges: false,
				WorkerPoolProcessor:  workerPoolProcessor,
				ExcludeTenantTags:    args.ExcludeTenantTags,
				ExcludeTenantTagSets: args.ExcludeTenantTagSets,
				Excluder:             converters.DefaultExcluder{},
				TagSetConverter:      &tagsetConverter,
			},
			TenantConverter: &tenantConverter,
			ProjectTriggerConverter: converters.ProjectTriggerConverter{
				Client: octopusClient,
			},
			VariableSetConverter:      &variableSetConverter,
			ChannelConverter:          channelConverter,
			RunbookConverter:          &runbookConverter,
			IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
			ExcludeCaCProjectSettings: args.ExcludeCaCProjectSettings,
			ExcludeAllRunbooks:        false,
			IgnoreProjectChanges:      args.IgnoreProjectChanges,
			IgnoreProjectGroupChanges: false,
			IgnoreProjectNameChanges:  false,
			ExcludeProjects:           args.ExcludeProjects,
			ExcludeProjectsRegex:      args.ExcludeProjectsRegex,
			ExcludeAllProjects:        args.ExcludeAllProjects,
			ExcludeProjectsExcept:     args.ExcludeProjectsExcept,
			DummySecretVariableValues: args.DummySecretVariableValues,
			DummySecretGenerator:      dummySecretGenerator,
			Excluder:                  converters.DefaultExcluder{},
			ErrGroup:                  &group,
		},
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
	}

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

	if args.Stateless {
		templateGenerator := generators.StepTemplateGenerator{}
		templateContent, err := templateGenerator.Generate(&dependencies, args.StepTemplateName, args.StepTemplateKey, args.StepTemplateDescription)

		if err != nil {
			return nil, err
		}

		return map[string]string{"step_template.json": string(templateContent[:])}, nil
	} else {
		return processResources(dependencies.Resources)
	}
}

func ConvertRunbookToTerraform(args args.Arguments) (map[string]string, error) {

	octopusClient := client.OctopusApiClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dummySecretGenerator := converters.DummySecret{}

	dependencies := data.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	environmentConverter := converters.EnvironmentConverter{Client: octopusClient}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:               octopusClient,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
	}

	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                    octopusClient,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		Excluder:                  converters.DefaultExcluder{},
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeProjectsExcept:     args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:      args.ExcludeProjectsRegex,
		ExcludeAllProjects:        args.ExcludeAllProjects,
		ExcludeProjects:           args.ExcludeProjects,
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         &tagsetConverter,
		ExcludeTenants:          args.ExcludeTenants,
		ExcludeTenantsRegex:     args.ExcludeTenantsRegex,
		ExcludeAllTenants:       args.ExcludeAllTenants,
		ExcludeTenantsExcept:    args.ExcludeTenantsExcept,
		ExcludeTenantsWithTags:  args.ExcludeTenantsWithTags,
		Excluder:                converters.DefaultExcluder{},
		ExcludeTenantTags:       args.ExcludeTenantTags,
		ExcludeTenantTagSets:    args.ExcludeTenantTagSets,
		ExcludeProjectsExcept:   args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:    args.ExcludeProjectsRegex,
		ExcludeAllProjects:      args.ExcludeAllProjects,
		ExcludeProjects:         args.ExcludeProjects,
	}

	accountConverter := converters.AccountConverter{
		Client:                    octopusClient,
		EnvironmentConverter:      environmentConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
	}

	feedConverter := converters.FeedConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
	}
	workerPoolConverter := converters.WorkerPoolConverter{Client: octopusClient}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  octopusClient,
	}

	projectConverter := &converters.ProjectConverter{
		LookupOnlyMode: true,
		Client:         octopusClient,
		Excluder:       converters.DefaultExcluder{},
	}

	runbookConverter := converters.RunbookConverter{
		Client: octopusClient,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: octopusClient,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:           feedConverter,
				AccountConverter:        accountConverter,
				WorkerPoolConverter:     workerPoolConverter,
				EnvironmentConverter:    environmentConverter,
				DetachProjectTemplates:  args.DetachProjectTemplates,
				WorkerPoolProcessor:     workerPoolProcessor,
				GitCredentialsConverter: gitCredentialsConverter,
			},
			IgnoreProjectChanges: args.IgnoreProjectChanges,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    args.ExcludeTenantTags,
			ExcludeTenantTagSets: args.ExcludeTenantTagSets,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		EnvironmentConverter:  environmentConverter,
		ExcludedRunbooks:      nil,
		ExcludeRunbooksRegex:  nil,
		ExcludeRunbooksExcept: nil,
		ExcludeAllRunbooks:    false,
		Excluder:              converters.DefaultExcluder{},
		IgnoreProjectChanges:  args.IgnoreProjectChanges,
		ProjectConverter:      projectConverter,
	}

	err := runbookConverter.ToHclByIdWithLookups(args.RunbookId, &dependencies)

	if err != nil {
		return nil, err
	}

	return processResources(dependencies.Resources)
}

func ConvertProjectToTerraform(args args.Arguments) (map[string]string, error) {

	octopusClient := client.OctopusApiClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dummySecretGenerator := converters.DummySecret{}

	dependencies := data.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	environmentConverter := converters.EnvironmentConverter{Client: octopusClient}
	lifecycleConverter := converters.LifecycleConverter{Client: octopusClient, EnvironmentConverter: environmentConverter}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		ExcludeAllGitCredentials:  args.ExcludeAllGitCredentials,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:               octopusClient,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
	}
	channelConverter := converters.ChannelConverter{
		Client:               octopusClient,
		LifecycleConverter:   lifecycleConverter,
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
		Excluder:             converters.DefaultExcluder{},
	}

	projectGroupConverter := converters.ProjectGroupConverter{Client: octopusClient}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                    octopusClient,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		Excluder:                  converters.DefaultExcluder{},
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeProjectsExcept:     args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:      args.ExcludeProjectsRegex,
		ExcludeAllProjects:        args.ExcludeAllProjects,
		ExcludeProjects:           args.ExcludeProjects,
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         &tagsetConverter,
		ExcludeTenants:          args.ExcludeTenants,
		ExcludeTenantsRegex:     args.ExcludeTenantsRegex,
		ExcludeAllTenants:       args.ExcludeAllTenants,
		ExcludeTenantsExcept:    args.ExcludeTenantsExcept,
		ExcludeTenantsWithTags:  args.ExcludeTenantsWithTags,
		Excluder:                converters.DefaultExcluder{},
		ExcludeTenantTags:       args.ExcludeTenantTags,
		ExcludeTenantTagSets:    args.ExcludeTenantTagSets,
		ExcludeProjectsExcept:   args.ExcludeProjectsExcept,
		ExcludeProjectsRegex:    args.ExcludeProjectsRegex,
		ExcludeAllProjects:      args.ExcludeAllProjects,
		ExcludeProjects:         args.ExcludeProjects,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: octopusClient}
	accountConverter := converters.AccountConverter{
		Client:                    octopusClient,
		EnvironmentConverter:      environmentConverter,
		TenantConverter:           &tenantConverter,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
	}
	certificateConverter := converters.CertificateConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
	}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                    octopusClient,
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                    octopusClient,
		MachinePolicyConverter:    machinePolicyConverter,
		EnvironmentConverter:      environmentConverter,
		ExcludeAllTargets:         args.ExcludeAllTargets,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           &tagsetConverter,
		ExcludeTargets:            args.ExcludeTargets,
		ExcludeTargetsRegex:       args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:      args.ExcludeTargetsExcept,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	feedConverter := converters.FeedConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
	}
	workerPoolConverter := converters.WorkerPoolConverter{Client: octopusClient}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            octopusClient,
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
		DummySecretGenerator:              dummySecretGenerator,
		ExcludeAllProjectVariables:        args.ExcludeAllProjectVariables,
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesExcept:     args.ExcludeProjectVariablesExcept,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeVariableEnvironmentScopes:  args.ExcludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              args.IgnoreProjectChanges || args.IgnoreProjectVariableChanges,
		Excluder:                          converters.DefaultExcluder{},
	}

	variableSetConverterForLibrary := converters.VariableSetConverter{
		Client:                            octopusClient,
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
		DummySecretGenerator:              dummySecretGenerator,
		ExcludeAllProjectVariables:        false,
		ExcludeProjectVariables:           nil,
		ExcludeProjectVariablesExcept:     nil,
		ExcludeProjectVariablesRegex:      nil,
		ExcludeVariableEnvironmentScopes:  args.ExcludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              false,
		Excluder:                          converters.DefaultExcluder{},
	}

	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                           octopusClient,
		VariableSetConverter:             &variableSetConverterForLibrary,
		Excluded:                         args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex:  args.ExcludeLibraryVariableSetsRegex,
		ExcludeLibraryVariableSetsExcept: args.ExcludeLibraryVariableSetsExcept,
		ExcludeAllLibraryVariableSets:    args.ExcludeAllLibraryVariableSets,
		DummySecretVariableValues:        args.DummySecretVariableValues,
		DummySecretGenerator:             dummySecretGenerator,
		Excluder:                         converters.DefaultExcluder{},
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  octopusClient,
	}

	runbookConverter := converters.RunbookConverter{
		Client: octopusClient,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: octopusClient,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:           feedConverter,
				AccountConverter:        accountConverter,
				WorkerPoolConverter:     workerPoolConverter,
				EnvironmentConverter:    environmentConverter,
				DetachProjectTemplates:  args.DetachProjectTemplates,
				WorkerPoolProcessor:     workerPoolProcessor,
				GitCredentialsConverter: gitCredentialsConverter,
			},
			IgnoreProjectChanges: args.IgnoreProjectChanges,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    args.ExcludeTenantTags,
			ExcludeTenantTagSets: args.ExcludeTenantTagSets,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		EnvironmentConverter:  environmentConverter,
		ProjectConverter:      nil,
		ExcludedRunbooks:      args.ExcludeRunbooks,
		ExcludeRunbooksRegex:  args.ExcludeRunbooksRegex,
		ExcludeRunbooksExcept: args.ExcludeRunbooksExcept,
		ExcludeAllRunbooks:    args.ExcludeAllRunbooks,
		Excluder:              converters.DefaultExcluder{},
		IgnoreProjectChanges:  args.IgnoreProjectChanges,
	}

	projectConverter := converters.ProjectConverter{
		ExcludeAllRunbooks:          args.ExcludeAllRunbooks,
		Client:                      octopusClient,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client: octopusClient,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:           feedConverter,
				AccountConverter:        accountConverter,
				WorkerPoolConverter:     workerPoolConverter,
				EnvironmentConverter:    environmentConverter,
				DetachProjectTemplates:  args.DetachProjectTemplates,
				WorkerPoolProcessor:     workerPoolProcessor,
				GitCredentialsConverter: gitCredentialsConverter,
			},
			IgnoreProjectChanges: args.IgnoreProjectChanges,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    args.ExcludeTenantTags,
			ExcludeTenantTagSets: args.ExcludeTenantTagSets,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: octopusClient,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
		ExcludeCaCProjectSettings: args.ExcludeCaCProjectSettings,
		RunbookConverter:          &runbookConverter,
		IgnoreProjectChanges:      args.IgnoreProjectChanges,
		IgnoreProjectGroupChanges: args.IgnoreProjectGroupChanges,
		IgnoreProjectNameChanges:  args.IgnoreProjectNameChanges,
		ExcludeProjects:           nil,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		Excluder:                  converters.DefaultExcluder{},
		ExcludeAllProjects:        false,
		ExcludeProjectsRegex:      nil,
		ExcludeProjectsExcept:     nil,
	}

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

	if args.Stateless {
		templateGenerator := generators.StepTemplateGenerator{}
		templateContent, err := templateGenerator.Generate(&dependencies, args.StepTemplateName, args.StepTemplateKey, args.StepTemplateDescription)

		if err != nil {
			return nil, err
		}

		return map[string]string{"step_template.json": string(templateContent[:])}, nil
	} else {
		return processResources(dependencies.Resources)
	}
}

// processResources creates a map of file names to file content
func processResources(resources []data.ResourceDetails) (map[string]string, error) {
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
