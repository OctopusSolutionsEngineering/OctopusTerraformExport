package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/converters"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/writers"
	"go.uber.org/zap"
	"os"
	"strings"
)

func main() {
	args, output, err := args.ParseArgs(os.Args[1:])

	if err == flag.ErrHelp {
		fmt.Println(output)
		os.Exit(2)
	} else if err != nil {
		fmt.Println("got error:", err)
		fmt.Println("output:\n", output)
		os.Exit(1)
	}

	if args.Url == "" {
		errorExit("You must specify the URL with the -url argument")
	}

	if args.ApiKey == "" {
		errorExit("You must specify the API key with the -apiKey argument")
	}

	projectId := args.ProjectId
	if args.ProjectName != "" {
		projectId, err = ConvertProjectNameToId(args.Url, args.Space, args.ApiKey, args.ProjectName)

		if err != nil {
			errorExit(err.Error())
		}
	}

	if projectId != "" {
		err = ConvertProjectToTerraform(
			args.Url,
			args.Space,
			args.ApiKey,
			args.Destination,
			args.Console,
			projectId,
			args.LookupProjectDependencies,
			args.IgnoreCacManagedValues,
			args.BackendBlock,
			args.DefaultSecretVariableValues,
			args.ProviderVersion,
			args.DetachProjectTemplates,
			args.ExcludeAllRunbooks,
			args.ExcludeRunbooks,
			args.ExcludeRunbooksRegex,
			args.ExcludeProvider,
			args.IncludeOctopusOutputVars,
			args.ExcludeLibraryVariableSets,
			args.ExcludeLibraryVariableSetsRegex,
			args.IgnoreProjectChanges,
			args.IgnoreProjectVariableChanges,
			args.ExcludeProjectVariables,
			args.ExcludeProjectVariablesRegex,
			args.ExcludeVariableEnvironmentScopes,
			args.IgnoreProjectGroupChanges,
			args.IgnoreProjectNameChanges,
			args.LookUpDefaultWorkerPools,
			args.ExcludeTenants,
			args.ExcludeAllTenants)
	} else {
		err = ConvertSpaceToTerraform(args.Url, args.Space, args.ApiKey, args.Destination, args.Console, args.DetachProjectTemplates)
	}

	if err != nil {
		errorExit(err.Error())
	}
}

func errorExit(message string) {
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
	client.GetAllResources("Projects", &collection, []string{"name", name})

	for _, p := range collection.Items {
		if p.Name == name {
			return p.Id, nil
		}
	}

	return "", errors.New("did not find project with name " + name)
}

func ConvertSpaceToTerraform(url string, space string, apiKey string, dest string, console bool, detachProjectTemplates bool) error {
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	environmentConverter := converters.EnvironmentConverter{Client: client}
	tenantVariableConverter := converters.TenantVariableConverter{Client: client}
	tagsetConverter := converters.TagSetConverter{Client: client}
	tenantConverter := converters.TenantConverter{
		Client:                  client,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
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
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
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
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{Client: client, VariableSetConverter: &variableSetConverter}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: false,
		Client:                  client,
	}

	runbookConverter := converters.RunbookConverter{
		Client: client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: client,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:          feedConverter,
				AccountConverter:       accountConverter,
				WorkerPoolConverter:    workerPoolConverter,
				EnvironmentConverter:   environmentConverter,
				DetachProjectTemplates: false,
				WorkerPoolProcessor:    workerPoolProcessor,
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
					FeedConverter:          feedConverter,
					AccountConverter:       accountConverter,
					WorkerPoolConverter:    workerPoolConverter,
					EnvironmentConverter:   environmentConverter,
					DetachProjectTemplates: false,
					WorkerPoolProcessor:    workerPoolProcessor,
				},
				IgnoreProjectChanges: false,
				WorkerPoolProcessor:  workerPoolProcessor,
			},
			TenantConverter: tenantConverter,
			ProjectTriggerConverter: converters.ProjectTriggerConverter{
				Client: client,
			},
			VariableSetConverter:   &variableSetConverter,
			ChannelConverter:       channelConverter,
			RunbookConverter:       &runbookConverter,
			IgnoreCacManagedValues: false,
			ExcludeAllRunbooks:     false,
			IgnoreProjectChanges:   false,
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

	dependencies := converters.ResourceDetailsCollection{}

	err := spaceConverter.ToHcl(&dependencies)

	if err != nil {
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(strutil.UnEscapeDollar(hcl), dest, console)

	return err
}

func ConvertProjectToTerraform(
	url string,
	space string,
	apiKey string,
	dest string,
	console bool,
	projectId string,
	lookupProjectDependencies bool,
	ignoreCacManagedSettings bool,
	terraformBackend string,
	defaultSecretVariableValues bool,
	providerVersion string,
	detachProjectTemplates bool,
	excludeRunbooks bool,
	excludedRunbooks args.ExcludeRunbooks,
	excludeRunbooksRegex args.ExcludeRunbooks,
	excludeProvider bool,
	includeOctopusOutputVars bool,
	excludedLibraryVariableSets args.ExcludeLibraryVariableSets,
	excludeLibraryVariableSetsRegex args.ExcludeLibraryVariableSets,
	ignoreProjectChanges bool,
	ignoreProjectVariableChanges bool,
	excludedVars args.ExcludeVariables,
	excludedVarsRegex args.ExcludeVariables,
	excludeVariableEnvironmentScopes args.ExcludeVariableEnvironmentScopes,
	ignoreProjectGroupChanges bool,
	ignoreProjectNameChanges bool,
	lookUpDefaultWorkerPools bool,
	excludeTenants args.ExcludeTenants,
	excludeAllTenants bool) error {

	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	dependencies := converters.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{
		TerraformBackend:         terraformBackend,
		ProviderVersion:          providerVersion,
		ExcludeProvider:          excludeProvider,
		IncludeOctopusOutputVars: includeOctopusOutputVars,
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
	tenantVariableConverter := converters.TenantVariableConverter{Client: client}
	tenantConverter := converters.TenantConverter{
		Client:                  client,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         tagsetConverter,
		ExcludeTenants:          excludeTenants,
		ExcludeAllTenants:       excludeAllTenants,
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
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
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
		IgnoreCacManagedValues:            ignoreCacManagedSettings,
		DefaultSecretVariableValues:       defaultSecretVariableValues,
		ExcludeProjectVariables:           excludedVars,
		ExcludeProjectVariablesRegex:      excludedVarsRegex,
		ExcludeVariableEnvironmentScopes:  excludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              ignoreProjectChanges || ignoreProjectVariableChanges,
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
		IgnoreCacManagedValues:            ignoreCacManagedSettings,
		DefaultSecretVariableValues:       defaultSecretVariableValues,
		ExcludeProjectVariables:           excludedVars,
		ExcludeProjectVariablesRegex:      excludedVarsRegex,
		ExcludeVariableEnvironmentScopes:  excludeVariableEnvironmentScopes,
		IgnoreProjectChanges:              false,
	}

	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:                          client,
		VariableSetConverter:            &variableSetConverterForLibrary,
		Excluded:                        excludedLibraryVariableSets,
		ExcludeLibraryVariableSetsRegex: excludeLibraryVariableSetsRegex,
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: lookUpDefaultWorkerPools,
		Client:                  client,
	}

	runbookConverter := converters.RunbookConverter{
		Client: client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: client,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:          feedConverter,
				AccountConverter:       accountConverter,
				WorkerPoolConverter:    workerPoolConverter,
				EnvironmentConverter:   environmentConverter,
				DetachProjectTemplates: detachProjectTemplates,
				WorkerPoolProcessor:    workerPoolProcessor,
			},
			IgnoreProjectChanges: ignoreProjectChanges,
			WorkerPoolProcessor:  workerPoolProcessor,
		},
		EnvironmentConverter: environmentConverter,
		ExcludedRunbooks:     excludedRunbooks,
		ExcludeRunbooksRegex: excludeRunbooksRegex,
		IgnoreProjectChanges: ignoreProjectChanges,
	}

	projectConverter := converters.ProjectConverter{
		ExcludeAllRunbooks:          excludeRunbooks,
		Client:                      client,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client: client,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:          feedConverter,
				AccountConverter:       accountConverter,
				WorkerPoolConverter:    workerPoolConverter,
				EnvironmentConverter:   environmentConverter,
				DetachProjectTemplates: detachProjectTemplates,
				WorkerPoolProcessor:    workerPoolProcessor,
			},
			IgnoreProjectChanges: ignoreProjectChanges,
			WorkerPoolProcessor:  workerPoolProcessor,
		},
		TenantConverter: tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: client,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		IgnoreCacManagedValues:    ignoreCacManagedSettings,
		RunbookConverter:          &runbookConverter,
		IgnoreProjectChanges:      ignoreProjectChanges,
		IgnoreProjectGroupChanges: ignoreProjectGroupChanges,
		IgnoreProjectNameChanges:  ignoreProjectNameChanges,
	}

	var err error
	if lookupProjectDependencies {
		err = projectConverter.ToHclLookupById(projectId, &dependencies)
	} else {
		err = projectConverter.ToHclById(projectId, &dependencies)
	}

	if err != nil {
		return err
	}

	hcl, err := processResources(dependencies.Resources)

	if err != nil {
		return err
	}

	err = writeFiles(strutil.UnEscapeDollar(hcl), dest, console)

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
