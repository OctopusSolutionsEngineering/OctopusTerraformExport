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

func main() {
	logger.BuildLogger()

	args, output, err := args.ParseArgs(os.Args[1:])

	if errors.Is(err, flag.ErrHelp) {
		zap.L().Error(output)
		os.Exit(2)
	} else if err != nil {
		zap.L().Error("got error: " + err.Error())
		zap.L().Error("output:\n" + output)
		os.Exit(1)
	}

	if args.Url == "" {
		errorExit("You must specify the URL with the -url argument")
	}

	if args.ApiKey == "" {
		errorExit("You must specify the API key with the -apiKey argument")
	}

	if args.ProjectName != "" {
		args.ProjectId, err = ConvertProjectNameToId(args.Url, args.Space, args.ApiKey, args.ProjectName)

		if err != nil {
			errorExit(err.Error())
		}
	}

	if args.ProjectId != "" {
		zap.L().Info("Exporting project " + args.ProjectId + " in space " + args.Space)
		err = ConvertProjectToTerraform(args)
	} else {
		zap.L().Info("Exporting space " + args.Space)
		err = ConvertSpaceToTerraform(args)
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
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	collection := octopus.GeneralCollection[octopus.Project]{}
	err := client.GetAllResources("Projects", &collection, []string{"name", name})

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

func ConvertSpaceToTerraform(args args.Arguments) error {
	client := client.OctopusClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dependencies := converters.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", &dependencies)

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_creation", &dependencies)

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	environmentConverter := converters.EnvironmentConverter{Client: client}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:               client,
		ExcludeTenants:       args.ExcludeTenants,
		ExcludeAllTenants:    args.ExcludeAllTenants,
		ExcludeTenantsExcept: args.ExcludeTenantsExcept,
		Excluder:             converters.DefaultExcluder{},
	}
	tagsetConverter := converters.TagSetConverter{Client: client}
	tenantConverter := converters.TenantConverter{
		Client:                  client,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
		ExcludeTenants:          args.ExcludeTenants,
		ExcludeAllTenants:       args.ExcludeAllTenants,
		ExcludeTenantsExcept:    args.ExcludeTenantsExcept,
		Excluder:                converters.DefaultExcluder{},
		ExcludeProjects:         args.ExcludeProjects,
	}
	accountConverter := converters.AccountConverter{
		Client:               client,
		EnvironmentConverter: machinePolicyConverter,
		TenantConverter:      tenantConverter}

	lifecycleConverter := converters.LifecycleConverter{Client: client, EnvironmentConverter: environmentConverter}
	gitCredentialsConverter := converters.GitCredentialsConverter{Client: client}
	channelConverter := converters.ChannelConverter{
		Client:             client,
		LifecycleConverter: lifecycleConverter,
	}

	projectGroupConverter := converters.ProjectGroupConverter{Client: client}

	certificateConverter := converters.CertificateConverter{Client: client}
	workerPoolConverter := converters.WorkerPoolConverter{Client: client}

	feedConverter := converters.FeedConverter{Client: client}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            client,
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
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                          client,
		VariableSetConverter:            &variableSetConverter,
		Excluded:                        args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex: args.ExcludeLibraryVariableSetsRegex,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  client,
	}

	runbookConverter := converters.RunbookConverter{
		Client: client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: client,
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
		},
		EnvironmentConverter: environmentConverter,
		ExcludedRunbooks:     nil,
		ExcludeRunbooksRegex: nil,
		IgnoreProjectChanges: false,
	}

	spaceConverter := converters.SpaceConverter{
		Client:                      client,
		AccountConverter:            accountConverter,
		EnvironmentConverter:        environmentConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		LifecycleConverter:          lifecycleConverter,
		WorkerPoolConverter:         workerPoolConverter,
		TagSetConverter:             tagsetConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		ProjectGroupConverter:       projectGroupConverter,
		ProjectConverter: converters.ProjectConverter{
			Client:                      client,
			LifecycleConverter:          lifecycleConverter,
			GitCredentialsConverter:     gitCredentialsConverter,
			LibraryVariableSetConverter: &libraryVariableSetConverter,
			ProjectGroupConverter:       projectGroupConverter,
			DeploymentProcessConverter: converters.DeploymentProcessConverter{
				Client: client,
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
			},
			TenantConverter: tenantConverter,
			ProjectTriggerConverter: converters.ProjectTriggerConverter{
				Client: client,
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
		},
		TenantConverter:                   tenantConverter,
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

func ConvertProjectToTerraform(args args.Arguments) error {

	client := client.OctopusClient{
		Url:    args.Url,
		Space:  args.Space,
		ApiKey: args.ApiKey,
	}

	dependencies := converters.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         args.BackendBlock,
		ProviderVersion:          args.ProviderVersion,
		ExcludeProvider:          args.ExcludeProvider,
		IncludeOctopusOutputVars: args.IncludeOctopusOutputVars,
	}.ToHcl("space_population", &dependencies)

	environmentConverter := converters.EnvironmentConverter{Client: client}
	lifecycleConverter := converters.LifecycleConverter{Client: client, EnvironmentConverter: environmentConverter}
	gitCredentialsConverter := converters.GitCredentialsConverter{Client: client}
	tagsetConverter := converters.TagSetConverter{Client: client}
	channelConverter := converters.ChannelConverter{
		Client:             client,
		LifecycleConverter: lifecycleConverter,
	}

	projectGroupConverter := converters.ProjectGroupConverter{Client: client}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:               client,
		ExcludeTenants:       args.ExcludeTenants,
		ExcludeAllTenants:    args.ExcludeAllTenants,
		ExcludeTenantsExcept: args.ExcludeTenantsExcept,
		Excluder:             converters.DefaultExcluder{},
	}
	tenantConverter := converters.TenantConverter{
		Client:                  client,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
		ExcludeTenants:          args.ExcludeTenants,
		ExcludeAllTenants:       args.ExcludeAllTenants,
		ExcludeTenantsExcept:    args.ExcludeTenantsExcept,
		Excluder:                converters.DefaultExcluder{},
		ExcludeProjects:         args.ExcludeProjects,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	accountConverter := converters.AccountConverter{
		Client:               client,
		EnvironmentConverter: environmentConverter,
		TenantConverter:      tenantConverter,
	}
	certificateConverter := converters.CertificateConverter{Client: client}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeAllTargets:      args.ExcludeAllTargets,
	}

	feedConverter := converters.FeedConverter{Client: client}
	workerPoolConverter := converters.WorkerPoolConverter{Client: client}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            client,
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
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeVariableEnvironmentScopes:  args.ExcludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              args.IgnoreProjectChanges || args.IgnoreProjectVariableChanges,
	}

	variableSetConverterForLibrary := converters.VariableSetConverter{
		Client:                            client,
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
		ExcludeProjectVariables:           args.ExcludeProjectVariables,
		ExcludeProjectVariablesRegex:      args.ExcludeProjectVariablesRegex,
		ExcludeVariableEnvironmentScopes:  args.ExcludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              false,
	}

	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                          client,
		VariableSetConverter:            &variableSetConverterForLibrary,
		Excluded:                        args.ExcludeLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex: args.ExcludeLibraryVariableSetsRegex,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: args.LookUpDefaultWorkerPools,
		Client:                  client,
	}

	runbookConverter := converters.RunbookConverter{
		Client: client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: client,
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
		},
		EnvironmentConverter: environmentConverter,
		ExcludedRunbooks:     args.ExcludeRunbooks,
		ExcludeRunbooksRegex: args.ExcludeRunbooksRegex,
		IgnoreProjectChanges: args.IgnoreProjectChanges,
	}

	projectConverter := converters.ProjectConverter{
		ExcludeAllRunbooks:          args.ExcludeAllRunbooks,
		Client:                      client,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client: client,
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
		},
		TenantConverter: tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: client,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		IgnoreCacManagedValues:    args.IgnoreCacManagedValues,
		RunbookConverter:          &runbookConverter,
		IgnoreProjectChanges:      args.IgnoreProjectChanges,
		IgnoreProjectGroupChanges: args.IgnoreProjectGroupChanges,
		IgnoreProjectNameChanges:  args.IgnoreProjectNameChanges,
		ExcludeProjects:           nil,
	}

	var err error
	if args.LookupProjectDependencies {
		err = projectConverter.ToHclLookupById(args.ProjectId, &dependencies)
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
