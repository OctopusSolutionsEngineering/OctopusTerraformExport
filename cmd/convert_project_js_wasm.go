package main

import (
	"fmt"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/converters"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"sort"
	"strings"
	"syscall/js"
)

func main() {
	c := make(chan bool)
	js.Global().Set("convertProject", convertProject())
	<-c
}

func convertProject() js.Func {
	return js.FuncOf(func(this js.Value, funcArgs []js.Value) any {
		if len(funcArgs) != 3 {
			fmt.Println("Must pass in url, space, and project slug")
		}

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]

			go func() {
				files, err := convertProjectToTerraform(funcArgs[0].String(), funcArgs[1].String(), funcArgs[2].String())

				if err != nil {
					reject.Invoke(err.Error())
				}

				hclBlob := ""

				for _, h := range strutil.UnEscapeDollar(files) {
					hclBlob += h + "\n"
				}

				resolve.Invoke(hclBlob)
			}()

			return nil
		})

		// Create and return the Promise object
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

func convertProjectToTerraform(url string, space string, projectId string) (map[string]string, error) {
	client := client.OctopusClient{
		Url:   url,
		Space: space,
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
		TagSetConverter:         tagsetConverter,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	accountConverter := converters.AccountConverter{Client: client, EnvironmentConverter: lifecycleConverter, TenantConverter: tenantConverter}
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
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{Client: client, VariableSetConverter: variableSetConverter}

	runbookConverter := converters.RunbookConverter{
		Client: client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client:               client,
			FeedConverter:        feedConverter,
			AccountConverter:     accountConverter,
			WorkerPoolConverter:  workerPoolConverter,
			EnvironmentConverter: environmentConverter,
		},
		EnvironmentConverter: environmentConverter,
	}

	err := converters.ProjectConverter{
		Client:                      client,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client:                 client,
			FeedConverter:          feedConverter,
			AccountConverter:       accountConverter,
			WorkerPoolConverter:    workerPoolConverter,
			EnvironmentConverter:   environmentConverter,
			DetachProjectTemplates: false,
		},
		TenantConverter: tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: client,
		},
		VariableSetConverter:   variableSetConverter,
		ChannelConverter:       channelConverter,
		RunbookConverter:       runbookConverter,
		IgnoreCacManagedValues: false,
	}.ToHclLookupById(projectId, &dependencies)

	if err != nil {
		return nil, err
	}

	return processJavaScriptResources(dependencies.Resources)
}

// processResources creates a map of file names to file content
func processJavaScriptResources(resources []converters.ResourceDetails) (map[string]string, error) {
	fileMap := map[string]string{}

	// Sort by resource type
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ResourceType < resources[j].ResourceType
	})

	for index, r := range resources {
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
			fileMap["tf"+fmt.Sprintln(index)+".tf"] = hcl
		}
	}

	return fileMap, nil
}
