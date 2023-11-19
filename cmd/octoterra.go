package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/converters"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/logger"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/writers"
	"go.uber.org/zap"
	"os"
	"strings"
)

var Version = "development"

func main() {
	logger.BuildLogger()

	parseArgs, output, err := args.ParseArgs(os.Args[1:])

	if errors.Is(err, flag.ErrHelp) {
		zap.L().Error(output)
		os.Exit(2)
	} else if err != nil {
		zap.L().Error("got error: " + err.Error())
		zap.L().Error("output:\n" + output)
		os.Exit(1)
	}

	if parseArgs.Version {
		zap.L().Info("Version: " + Version)
		os.Exit(0)
	}

	if parseArgs.Url == "" {
		errorExit("You must specify the URL with the -url argument")
	}

	if parseArgs.ApiKey == "" {
		errorExit("You must specify the API key with the -apiKey argument")
	}

	if parseArgs.RunbookName != "" && parseArgs.ProjectName == "" && parseArgs.ProjectId == "" {
		errorExit("runbookName requires either projectId or projectName to be set")
	}

	if parseArgs.ProjectName != "" {
		parseArgs.ProjectId, err = ConvertProjectNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, parseArgs.ProjectName)

		if err != nil {
			errorExit(err.Error())
		}
	}

	if parseArgs.RunbookName != "" {
		parseArgs.RunbookId, err = ConvertRunbookNameToId(parseArgs.Url, parseArgs.Space, parseArgs.ApiKey, parseArgs.ProjectId, parseArgs.RunbookName)

		if err != nil {
			errorExit(err.Error())
		}
	}

	if parseArgs.RunbookId != "" {
		zap.L().Info("Exporting runbook " + parseArgs.RunbookId + " in space " + parseArgs.Space)
		err = ConvertRunbookToTerraform(parseArgs)
	} else if parseArgs.ProjectId != "" {
		zap.L().Info("Exporting project " + parseArgs.ProjectId + " in space " + parseArgs.Space)
		err = ConvertProjectToTerraform(parseArgs)
	} else {
		zap.L().Info("Exporting space " + parseArgs.Space)
		err = ConvertSpaceToTerraform(parseArgs)
	}

	if err != nil {
		errorExit(err.Error())
	}
}

func errorExit(message string) {
	if len(message) == 0 {
		message = "No error message provided"
	}
	zap.L().Error(message)
	os.Exit(1)
}

func ConvertProjectNameToId(url string, space string, apiKey string, name string) (string, error) {
	octopusClient := client.OctopusClient{
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
	octopusClient := client.OctopusClient{
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

func ConvertSpaceToTerraform(args args.Arguments) error {
	octopusClient := client.OctopusClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dependencies := converters.ResourceDetailsCollection{}

	dummySecretGenerator := converters.DummySecret{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", true, &dependencies)

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_creation", false, &dependencies)

	machinePolicyConverter := converters.MachinePolicyConverter{Client: octopusClient}
	environmentConverter := converters.EnvironmentConverter{Client: octopusClient}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                    octopusClient,
		ExcludeTenants:            args.ExcludeTenants,
		ExcludeTenantsWithTags:    args.ExcludeTenantsWithTags,
		ExcludeAllTenants:         args.ExcludeAllTenants,
		ExcludeTenantsExcept:      args.ExcludeTenantsExcept,
		Excluder:                  converters.DefaultExcluder{},
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
	}
	tagsetConverter := converters.TagSetConverter{
		Client:               octopusClient,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
	}

	lifecycleConverter := converters.LifecycleConverter{
		Client:               octopusClient,
		EnvironmentConverter: environmentConverter,
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
	}
	channelConverter := converters.ChannelConverter{
		Client:               octopusClient,
		LifecycleConverter:   lifecycleConverter,
		ExcludeTenantTags:    args.ExcludeTenantTags,
		ExcludeTenantTagSets: args.ExcludeTenantTagSets,
		Excluder:             converters.DefaultExcluder{},
	}

	projectGroupConverter := converters.ProjectGroupConverter{Client: octopusClient}

	certificateConverter := converters.CertificateConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           tagsetConverter,
	}
	workerPoolConverter := converters.WorkerPoolConverter{Client: octopusClient}

	feedConverter := converters.FeedConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
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
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 octopusClient,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
		ExcludeTenantTags:      args.ExcludeTenantTags,
		ExcludeTenantTagSets:   args.ExcludeTenantTagSets,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
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
		TagSetConverter:        tagsetConverter,
		ExcludeTargets:         args.ExcludeTargets,
		ExcludeTargetsRegex:    args.ExcludeTargetsRegex,
		ExcludeTargetsExcept:   args.ExcludeTargetsExcept,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            octopusClient,
		ChannelConverter:                  channelConverter,
		EnvironmentConverter:              environmentConverter,
		TagSetConverter:                   tagsetConverter,
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
			IgnoreProjectChanges: false,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    args.ExcludeTenantTags,
			ExcludeTenantTagSets: args.ExcludeTenantTagSets,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      tagsetConverter,
		},
		EnvironmentConverter:  environmentConverter,
		ExcludedRunbooks:      nil,
		ExcludeRunbooksRegex:  nil,
		Excluder:              converters.DefaultExcluder{},
		ExcludeRunbooksExcept: nil,
		ExcludeAllRunbooks:    false,
		ProjectConverter:      nil,
		IgnoreProjectChanges:  false,
	}

	spaceConverter := converters.SpaceConverter{
		Client:                      octopusClient,
		AccountConverter:            accountConverter,
		EnvironmentConverter:        environmentConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		LifecycleConverter:          lifecycleConverter,
		WorkerPoolConverter:         workerPoolConverter,
		TagSetConverter:             tagsetConverter,
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
				TagSetConverter:      tagsetConverter,
			},
			TenantConverter: &tenantConverter,
			ProjectTriggerConverter: converters.ProjectTriggerConverter{
				Client: octopusClient,
			},
			VariableSetConverter:      &variableSetConverter,
			ChannelConverter:          channelConverter,
			RunbookConverter:          &runbookConverter,
			IgnoreCacManagedValues:    false,
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
	}

	err := spaceConverter.ToHcl(&dependencies)

	if err != nil {
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(strutil.UnEscapeDollar(hcl), args.Destination, args.Console)

	return err
}

func ConvertRunbookToTerraform(args args.Arguments) error {

	octopusClient := client.OctopusClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dummySecretGenerator := converters.DummySecret{}

	dependencies := converters.ResourceDetailsCollection{}

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
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
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
			TagSetConverter:      tagsetConverter,
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
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(strutil.UnEscapeDollar(hcl), args.Destination, args.Console)

	return err
}

func ConvertProjectToTerraform(args args.Arguments) error {

	octopusClient := client.OctopusClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dummySecretGenerator := converters.DummySecret{}

	dependencies := converters.ResourceDetailsCollection{}

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
	}
	tenantConverter := converters.TenantConverter{
		Client:                  octopusClient,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
	}
	certificateConverter := converters.CertificateConverter{
		Client:                    octopusClient,
		DummySecretVariableValues: args.DummySecretVariableValues,
		DummySecretGenerator:      dummySecretGenerator,
		ExcludeTenantTags:         args.ExcludeTenantTags,
		ExcludeTenantTagSets:      args.ExcludeTenantTagSets,
		Excluder:                  converters.DefaultExcluder{},
		TagSetConverter:           tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:           tagsetConverter,
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
		TagSetConverter:        tagsetConverter,
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
		TagSetConverter:                   tagsetConverter,
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
		TagSetConverter:                   tagsetConverter,
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
			TagSetConverter:      tagsetConverter,
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
			TagSetConverter:      tagsetConverter,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: octopusClient,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
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

	var err error
	if args.LookupProjectDependencies {
		err = projectConverter.ToHclByIdWithLookups(args.ProjectId, &dependencies)
	} else {
		err = projectConverter.ToHclById(args.ProjectId, &dependencies)
	}

	if err != nil {
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(strutil.UnEscapeDollar(hcl), args.Destination, args.Console)

	return err
}

// processResources creates a map of file names to file content
func processResources(resources []converters.ResourceDetails) (map[string]string, error) {
	fileMap := map[string]string{}

	for _, r := range resources {
		// Some resources are already resolved by their parent, but exist in the resource details map as a lookup.
		// In these cases, ToHclByProjectId is nil.
		if r.ToHcl == nil {
			continue
		}

		hcl, err := r.ToHcl()

		if err != nil {
			return nil, err
		}

		if len(strings.TrimSpace(hcl)) != 0 {
			fileMap[r.FileName] = hcl
		}
	}

	return fileMap, nil
}

func writeFiles(files map[string]string, dest string, console bool) error {
	if dest != "" {
		writer := writers.NewFileWriter(dest)
		_, err := writer.Write(files)
		if err != nil {
			return err
		}
	}

	if console || dest == "" {
		consoleWriter := writers.ConsoleWriter{}
		output, err := consoleWriter.Write(files)
		if err != nil {
			return err
		}
		fmt.Println(output)
	}

	return nil
}
