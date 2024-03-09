package main

import (
	"encoding/json"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/converters"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"go.uber.org/zap"
	"sort"
	"strings"
	"syscall/js"
)

// This is the entrypoint of a WASM library that can be embedded in a web page to convert an
// Octopus project to HCL in the browser. See the wasm/violentmonkey.js file for an example where
// this WASM application is used.
func main() {
	c := make(chan bool)
	js.Global().Set("convertProject", convertProject())
	js.Global().Set("convertSpace", convertSpace())
	<-c
}

func convertSpace() js.Func {
	return js.FuncOf(func(this js.Value, funcArgs []js.Value) any {
		if len(funcArgs) < 2 {
			zap.L().Error("Must pass in url, space")
		}

		handler := js.FuncOf(func(this js.Value, jsargs []js.Value) interface{} {
			resolve := jsargs[0]
			reject := jsargs[1]

			arguments := args.Arguments{
				Url:                              funcArgs[0].String(),
				Space:                            funcArgs[1].String(),
				ExcludeAllProjects:               funcArgs[2].Bool(),
				ExcludeProjectsExcept:            strings.Split(funcArgs[3].String(), ","),
				ExcludeAllTargets:                funcArgs[4].Bool(),
				ExcludeTargetsExcept:             strings.Split(funcArgs[5].String(), ","),
				ExcludeAllRunbooks:               funcArgs[6].Bool(),
				ExcludeRunbooksExcept:            strings.Split(funcArgs[7].String(), ","),
				ExcludeAllLibraryVariableSets:    funcArgs[8].Bool(),
				ExcludeLibraryVariableSetsExcept: strings.Split(funcArgs[9].String(), ","),
				ExcludeAllTenants:                funcArgs[10].Bool(),
				ExcludeTenantsExcept:             strings.Split(funcArgs[11].String(), ","),
			}

			argsJson, _ := json.Marshal(arguments)
			zap.L().Info(string(argsJson))

			go func() {
				dependencies, err := entry.ConvertSpaceToTerraform(arguments)

				if err != nil {
					reject.Invoke(err.Error())
				}

				files, err := processJavaScriptResources(dependencies.Resources)

				if err != nil {
					reject.Invoke(err.Error())
				}

				hclBlob := ""

				for _, h := range strutil.UnEscapeDollarInMap(files) {
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

func convertProject() js.Func {
	return js.FuncOf(func(this js.Value, funcArgs []js.Value) any {
		if len(funcArgs) != 3 {
			zap.L().Error("Must pass in url, space, and project slug")
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

				for _, h := range strutil.UnEscapeDollarInMap(files) {
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
	client := client.OctopusApiClient{
		Url:   url,
		Space: space,
	}

	dependencies := data.ResourceDetailsCollection{}

	converters.TerraformProviderGenerator{}.ToHcl("space_population", true, &dependencies)

	environmentConverter := converters.EnvironmentConverter{Client: client}
	lifecycleConverter := converters.LifecycleConverter{Client: client, EnvironmentConverter: environmentConverter}
	gitCredentialsConverter := converters.GitCredentialsConverter{Client: client}
	tagsetConverter := converters.TagSetConverter{Client: client}
	channelConverter := converters.ChannelConverter{
		Client:               client,
		LifecycleConverter:   lifecycleConverter,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
	}

	projectGroupConverter := converters.ProjectGroupConverter{Client: client}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:   client,
		Excluder: converters.DefaultExcluder{},
	}
	tenantConverter := converters.TenantConverter{
		Client:                  client,
		TenantVariableConverter: tenantVariableConverter,
		EnvironmentConverter:    environmentConverter,
		TagSetConverter:         &tagsetConverter,
		Excluder:                converters.DefaultExcluder{},
	}

	machinePolicyConverter := converters.MachinePolicyConverter{Client: client}
	accountConverter := converters.AccountConverter{
		Client:               client,
		EnvironmentConverter: lifecycleConverter,
		TenantConverter:      &tenantConverter,
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
		Excluder:             converters.DefaultExcluder{},
		TagSetConverter:      &tagsetConverter,
	}
	certificateConverter := converters.CertificateConverter{
		Client:               client,
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
		Excluder:             converters.DefaultExcluder{},
		TagSetConverter:      &tagsetConverter,
	}
	workerPoolConverter := converters.WorkerPoolConverter{Client: client}
	feedConverter := converters.FeedConverter{Client: client}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	sshTargetConverter := converters.SshTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		Client:                 client,
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		Excluder:               converters.DefaultExcluder{},
		TagSetConverter:        &tagsetConverter,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            client,
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
		Excluder:                          converters.DefaultExcluder{},
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:               client,
		VariableSetConverter: &variableSetConverter,
		Excluder:             converters.DefaultExcluder{},
	}

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
				DetachProjectTemplates: true,
				WorkerPoolProcessor:    workerPoolProcessor,
			},
			IgnoreProjectChanges: false,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    nil,
			ExcludeTenantTagSets: nil,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		EnvironmentConverter:  environmentConverter,
		ExcludedRunbooks:      nil,
		ExcludeRunbooksRegex:  nil,
		IgnoreProjectChanges:  false,
		ExcludeRunbooksExcept: nil,
		Excluder:              converters.DefaultExcluder{},
		ExcludeAllRunbooks:    false,
	}

	err := (&converters.ProjectConverter{
		ExcludeAllRunbooks:          false,
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
				DetachProjectTemplates: true,
				WorkerPoolProcessor:    workerPoolProcessor,
			},
			IgnoreProjectChanges: false,
			WorkerPoolProcessor:  workerPoolProcessor,
			ExcludeTenantTags:    nil,
			ExcludeTenantTagSets: nil,
			Excluder:             converters.DefaultExcluder{},
			TagSetConverter:      &tagsetConverter,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client: client,
		},
		VariableSetConverter:      &variableSetConverter,
		ChannelConverter:          channelConverter,
		RunbookConverter:          &runbookConverter,
		IgnoreCacManagedValues:    false,
		ExcludeCaCProjectSettings: false,
		Excluder:                  converters.DefaultExcluder{},
	}).ToHclByIdWithLookups(projectId, &dependencies)

	if err != nil {
		return nil, err
	}

	return processJavaScriptResources(dependencies.Resources)
}

// processResources creates a map of file names to file content
func processJavaScriptResources(resources []data.ResourceDetails) (map[string]string, error) {
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
