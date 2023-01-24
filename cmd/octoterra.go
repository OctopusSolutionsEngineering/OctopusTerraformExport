package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/converters"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/strutil"
	"github.com/mcasperson/OctopusTerraformExport/internal/writers"
	"os"
	"strings"
)

func main() {
	url, space, apiKey, dest, console, projectId, projectName := parseUrl()

	var err error = nil

	if projectName != "" {
		projectId, err := convertProjectNameToId(url, space, apiKey, projectName)

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		err = ConvertProjectToTerraform(url, space, apiKey, dest, console, projectId)
	} else if projectId != "" {
		err = ConvertProjectToTerraform(url, space, apiKey, dest, console, projectId)
	} else {
		err = ConvertSpaceToTerraform(url, space, apiKey, dest, console)
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func convertProjectNameToId(url string, space string, apiKey string, name string) (string, error) {
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

func ConvertSpaceToTerraform(url string, space string, apiKey string, dest string, console bool) error {
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	accountConverter := converters.AccountConverter{Client: client}
	environmentConverter := converters.EnvironmentConverter{Client: client}
	tagsetConverter := converters.TagSetConverter{Client: client}
	lifecycleConverter := converters.LifecycleConverter{Client: client, EnvironmentConverter: environmentConverter}
	gitCredentialsConverter := converters.GitCredentialsConverter{Client: client}
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
	}
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
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{Client: client, VariableSetConverter: variableSetConverter}

	spaceConverter := converters.SpaceConverter{
		Client:                      client,
		AccountConverter:            accountConverter,
		EnvironmentConverter:        environmentConverter,
		LibraryVariableSetConverter: libraryVariableSetConverter,
		LifecycleConverter:          lifecycleConverter,
		WorkerPoolConverter:         workerPoolConverter,
		TagSetConverter:             tagsetConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		ProjectGroupConverter:       projectGroupConverter,
		ProjectConverter: converters.ProjectConverter{
			Client:                      client,
			LifecycleConverter:          lifecycleConverter,
			GitCredentialsConverter:     gitCredentialsConverter,
			LibraryVariableSetConverter: libraryVariableSetConverter,
			ProjectGroupConverter:       projectGroupConverter,
			DeploymentProcessConverter: converters.DeploymentProcessConverter{
				Client: client,
			},
			TenantConverter: tenantConverter,
			ProjectTriggerConverter: converters.ProjectTriggerConverter{
				Client: client,
			},
			VariableSetConverter: variableSetConverter,
			ChannelConverter:     channelConverter,
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

func ConvertProjectToTerraform(url string, space string, apiKey string, dest string, console bool, projectId string) error {
	client := client.OctopusClient{
		Url:    url,
		Space:  space,
		ApiKey: apiKey,
	}

	dependencies := converters.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{}.ToHcl("space_population", &dependencies)

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
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	accountConverter := converters.AccountConverter{Client: client}
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
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{Client: client, VariableSetConverter: variableSetConverter}

	err := converters.ProjectConverter{
		Client:                      client,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client: client,
		},
		TenantConverter: tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: client,
		},
		VariableSetConverter: variableSetConverter,
		ChannelConverter:     channelConverter,
	}.ToHclById(projectId, &dependencies)

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

func parseUrl() (string, string, string, string, bool, string, string) {
	var url string
	flag.StringVar(&url, "url", "", "The Octopus URL e.g. https://myinstance.octopus.app")

	var space string
	flag.StringVar(&space, "space", "", "The Octopus space name or ID")

	var apiKey string
	flag.StringVar(&apiKey, "apiKey", "", "The Octopus api key")

	var dest string
	flag.StringVar(&dest, "dest", "", "The directory to place the Terraform files in")

	var console bool
	flag.BoolVar(&console, "console", false, "Dump Terraform files to the console")

	var projectId string
	flag.StringVar(&projectId, "projectId", "", "Limit the export to a single project")

	var projectName string
	flag.StringVar(&projectName, "projectName", "", "Limit the export to a single project")

	flag.Parse()

	return url, space, apiKey, dest, console, projectId, projectName
}

func writeFiles(files map[string]string, dest string, console bool) error {
	writer := writers.NewFileWriter(dest)
	output, err := writer.Write(files)
	if err != nil {
		return err
	}
	fmt.Println(output)

	if console {
		consoleWriter := writers.ConsoleWriter{}
		output, err = consoleWriter.Write(files)
		if err != nil {
			return err
		}
		fmt.Println(output)
	}

	return nil
}
