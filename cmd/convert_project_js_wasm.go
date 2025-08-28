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

func getStringArg(funcArgs []js.Value, index int) string {
	if len(funcArgs) <= index {
		return ""
	}

	return funcArgs[index].String()
}

func getBoolArg(funcArgs []js.Value, index int) bool {
	if len(funcArgs) <= index {
		return false
	}

	return funcArgs[index].Bool()
}

func convertSpace() js.Func {
	return js.FuncOf(func(this js.Value, funcArgs []js.Value) any {
		if len(funcArgs) < 2 {
			zap.L().Error("Must pass url and space")
			return nil
		}

		handler := js.FuncOf(func(this js.Value, jsargs []js.Value) interface{} {
			resolve := jsargs[0]
			reject := jsargs[1]

			arguments := args.Arguments{
				Url:                              funcArgs[0].String(),
				Space:                            funcArgs[1].String(),
				ExcludeAllProjects:               funcArgs[2].Bool(),
				ExcludeProjectsExcept:            strings.Split(getStringArg(funcArgs, 3), ","),
				ExcludeAllTargets:                getBoolArg(funcArgs, 4),
				ExcludeTargetsExcept:             strings.Split(getStringArg(funcArgs, 5), ","),
				ExcludeAllRunbooks:               getBoolArg(funcArgs, 6),
				ExcludeRunbooksExcept:            strings.Split(getStringArg(funcArgs, 7), ","),
				ExcludeAllLibraryVariableSets:    getBoolArg(funcArgs, 8),
				ExcludeLibraryVariableSetsExcept: strings.Split(getStringArg(funcArgs, 9), ","),
				ExcludeAllTenants:                getBoolArg(funcArgs, 10),
				ExcludeTenantsExcept:             strings.Split(getStringArg(funcArgs, 11), ","),
				ExcludeAllEnvironments:           getBoolArg(funcArgs, 12),
				ExcludeEnvironmentsExcept:        strings.Split(getStringArg(funcArgs, 13), ","),
				ExcludeAllFeeds:                  getBoolArg(funcArgs, 14),
				ExcludeFeedsExcept:               strings.Split(getStringArg(funcArgs, 15), ","),
				ExcludeAllAccounts:               getBoolArg(funcArgs, 16),
				ExcludeAccountsExcept:            strings.Split(getStringArg(funcArgs, 17), ","),
				ExcludeAllCertificates:           getBoolArg(funcArgs, 18),
				ExcludeCertificatesExcept:        strings.Split(getStringArg(funcArgs, 19), ","),
				ExcludeAllLifecycles:             getBoolArg(funcArgs, 20),
				ExcludeLifecyclesExcept:          strings.Split(getStringArg(funcArgs, 21), ","),
				ExcludeAllWorkerpools:            getBoolArg(funcArgs, 22),
				ExcludeWorkerpoolsExcept:         strings.Split(getStringArg(funcArgs, 23), ","),
				ExcludeAllMachinePolicies:        getBoolArg(funcArgs, 24),
				ExcludeMachinePoliciesExcept:     strings.Split(getStringArg(funcArgs, 25), ","),
				ExcludeAllTenantTagSets:          getBoolArg(funcArgs, 26),
				ExcludeTenantTagSetsExcept:       strings.Split(getStringArg(funcArgs, 27), ","),
				ExcludeAllProjectGroups:          getBoolArg(funcArgs, 28),
				ExcludeProjectGroupsExcept:       strings.Split(getStringArg(funcArgs, 29), ","),
				ExcludeAllSteps:                  getBoolArg(funcArgs, 30),
				ExcludeStepsExcept:               strings.Split(getStringArg(funcArgs, 31), ","),
				ExcludeAllProjectVariables:       getBoolArg(funcArgs, 32),
				ExcludeProjectVariablesExcept:    strings.Split(getStringArg(funcArgs, 33), ","),
				ExcludeAllTenantVariables:        getBoolArg(funcArgs, 34),
				ExcludeTenantVariablesExcept:     strings.Split(getStringArg(funcArgs, 35), ","),
				ExcludeProvider:                  true,
				LimitAttributeLength:             100,
				IgnoreInvalidExcludeExcept:       true,
				ExcludeTerraformVariables:        true,
				ExcludeSpaceCreation:             true,
				IncludeIds:                       true,
				IncludeSpaceInPopulation:         true,
				IncludeDefaultChannel:            true,
				// We exclude targets with no environments if some environments were specifically mentioned
				// For example, if the environment "Prod" was mentioned, getBoolArg(funcArgs, 12) would be false,
				// and we want to limit targets to just that environment.
				ExcludeTargetsWithNoEnvironments: !getBoolArg(funcArgs, 12),
			}

			argsJson, _ := json.Marshal(arguments)
			zap.L().Info(string(argsJson))

			go func() {
				if err := arguments.ValidateExcludeExceptArgs(); err != nil {
					reject.Invoke(err.Error())
				}

				dependencies, err := entry.ConvertSpaceToTerraform(arguments)

				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				files, err := processJavaScriptResources(dependencies.Resources)

				if err != nil {
					reject.Invoke(err.Error())
					return
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

	stepTemplateConverter := StepTemplateConverter{}

	tenantCommonVariableProcessor := converters.TenantCommonVariableProcessor{
		Excluder:                     converters.DefaultExcluder{},
		ExcludeAllProjects:           false,
		ExcludeAllTenantVariables:    false,
		ExcludeTenantVariables:       nil,
		ExcludeTenantVariablesExcept: nil,
		ExcludeTenantVariablesRegex:  nil,
	}

	tenantProjectVariableConverter := converters.TenantProjectVariableConverter{
		Excluder:                     onverters.DefaultExcluder{},
		ExcludeAllProjects:           false,
		ExcludeAllTenantVariables:    false,
		ExcludeTenantVariables:       nil,
		ExcludeTenantVariablesExcept: nil,
		ExcludeTenantVariablesRegex:  nil,
		DummySecretVariableValues:    false,
		DummySecretGenerator:         nil,
	}

	tenantProjectConverter := converters.TenantProjectConverter{
		IncludeSpaceInPopulation: false,
		ErrGroup:                 nil,
		ExcludeTenantTagSets:     nil,
		ExcludeTenantTags:        nil,
		ExcludeTenants:           nil,
		ExcludeTenantsRegex:      nil,
		ExcludeTenantsWithTags:   nil,
		ExcludeTenantsExcept:     nil,
		ExcludeAllTenants:        false,
		Excluder:                 nil,
		Client:                   nil,
	}

	environmentConverter := converters.EnvironmentConverter{
		Client:   &client,
		Excluder: converters.DefaultExcluder{},
	}
	lifecycleConverter := converters.LifecycleConverter{
		Client:               &client,
		EnvironmentConverter: environmentConverter,
		Excluder:             converters.DefaultExcluder{},
	}
	gitCredentialsConverter := converters.GitCredentialsConverter{Client: &client}
	tagsetConverter := converters.TagSetConverter{
		Client:   &client,
		Excluder: converters.DefaultExcluder{},
	}
	channelConverter := converters.ChannelConverter{
		Client:               &client,
		LifecycleConverter:   lifecycleConverter,
		Excluder:             converters.DefaultExcluder{},
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
		IgnoreCacErrors:      false,
	}

	projectGroupConverter := converters.ProjectGroupConverter{
		Client:   &client,
		Excluder: converters.DefaultExcluder{},
	}
	tenantVariableConverter := converters.TenantVariableConverter{
		Client:                         &client,
		ExcludeTenants:                 nil,
		ExcludeTenantsWithTags:         nil,
		ExcludeTenantsExcept:           nil,
		ExcludeAllTenants:              false,
		Excluder:                       converters.DefaultExcluder{},
		DummySecretVariableValues:      false,
		DummySecretGenerator:           nil,
		ExcludeProjects:                nil,
		ExcludeProjectsExcept:          nil,
		ExcludeProjectsRegex:           nil,
		ExcludeAllProjects:             false,
		ErrGroup:                       nil,
		ExcludeAllTenantVariables:      false,
		ExcludeTenantVariables:         nil,
		ExcludeTenantVariablesExcept:   nil,
		ExcludeTenantVariablesRegex:    nil,
		TenantCommonVariableProcessor:  tenantCommonVariableProcessor,
		TenantProjectVariableConverter: tenantProjectVariableConverter,
	}
	tenantConverter := converters.TenantConverter{
		Client:                   &client,
		TenantVariableConverter:  tenantVariableConverter,
		EnvironmentConverter:     environmentConverter,
		TagSetConverter:          &tagsetConverter,
		ExcludeTenantTagSets:     nil,
		ExcludeTenantTags:        nil,
		ExcludeTenants:           nil,
		ExcludeTenantsRegex:      nil,
		ExcludeTenantsWithTags:   nil,
		ExcludeTenantsExcept:     nil,
		ExcludeAllTenants:        false,
		Excluder:                 converters.DefaultExcluder{},
		ExcludeProjects:          nil,
		ExcludeProjectsExcept:    nil,
		ExcludeProjectsRegex:     nil,
		ExcludeAllProjects:       false,
		ErrGroup:                 nil,
		IncludeIds:               false,
		LimitResourceCount:       0,
		IncludeSpaceInPopulation: false,
		GenerateImportScripts:    false,
		TenantProjectConverter:   tenantProjectConverter,
	}

	machinePolicyConverter := converters.MachinePolicyConverter{
		Client:                       &client,
		ErrGroup:                     nil,
		ExcludeMachinePolicies:       nil,
		ExcludeMachinePoliciesRegex:  nil,
		ExcludeMachinePoliciesExcept: nil,
		ExcludeAllMachinePolicies:    false,
		Excluder:                     converters.DefaultExcluder{},
	}
	accountConverter := converters.AccountConverter{
		Client:               &client,
		EnvironmentConverter: lifecycleConverter,
		TenantConverter:      &tenantConverter,
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
		Excluder:             converters.DefaultExcluder{},
		TagSetConverter:      &tagsetConverter,
	}
	certificateConverter := converters.CertificateConverter{
		Client:               &client,
		ExcludeTenantTags:    nil,
		ExcludeTenantTagSets: nil,
		Excluder:             converters.DefaultExcluder{},
		TagSetConverter:      &tagsetConverter,
	}
	workerPoolConverter := converters.WorkerPoolConverter{
		Client:                   &client,
		ErrGroup:                 nil,
		ExcludeWorkerpools:       nil,
		ExcludeWorkerpoolsRegex:  nil,
		ExcludeWorkerpoolsExcept: nil,
		ExcludeAllWorkerpools:    false,
		Excluder:                 converters.DefaultExcluder{},
	}
	feedConverter := converters.FeedConverter{
		Client:   &client,
		Excluder: converters.DefaultExcluder{},
	}

	kubernetesTargetConverter := converters.KubernetesTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		CertificateConverter:   certificateConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	sshTargetConverter := converters.SshTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	listeningTargetConverter := converters.ListeningTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	pollingTargetConverter := converters.PollingTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	cloudRegionTargetConverter := converters.CloudRegionTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	offlineDropTargetConverter := converters.OfflineDropTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	azureCloudServiceTargetConverter := converters.AzureCloudServiceTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	azureServiceFabricTargetConverter := converters.AzureServiceFabricTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	azureWebAppTargetConverter := converters.AzureWebAppTargetConverter{
		TargetConverter: converters.TargetConverter{
			Client:                           &client,
			Excluder:                         converters.DefaultExcluder{},
			ExcludeEnvironments:              nil,
			ExcludeEnvironmentsRegex:         nil,
			ExcludeEnvironmentsExcept:        nil,
			ExcludeAllEnvironments:           false,
			ExcludeTargetsWithNoEnvironments: false,
		},
		MachinePolicyConverter: machinePolicyConverter,
		AccountConverter:       accountConverter,
		EnvironmentConverter:   environmentConverter,
		ExcludeTenantTags:      nil,
		ExcludeTenantTagSets:   nil,
		TagSetConverter:        &tagsetConverter,
	}

	variableSetConverter := converters.VariableSetConverter{
		Client:                            &client,
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
		IgnoreCacManagedValues:            false,
		DefaultSecretVariableValues:       false,
		DummySecretVariableValues:         false,
		ExcludeAllProjectVariables:        false,
		ExcludeProjectVariables:           nil,
		ExcludeProjectVariablesExcept:     nil,
		ExcludeProjectVariablesRegex:      nil,
		ExcludeTenantTagSets:              nil,
		ExcludeTenantTags:                 nil,
		IgnoreProjectChanges:              false,
		DummySecretGenerator:              nil,
		TerraformVariableWriter:           nil,
		Excluder:                          converters.DefaultExcluder{},
		ErrGroup:                          nil,
		ExcludeTerraformVariables:         false,
		LimitAttributeLength:              0,
		StatelessAdditionalParams:         nil,
		GenerateImportScripts:             false,
		EnvironmentFilter: converters.EnvironmentFilter{
			Client:                           &octopusClient,
			ExcludeVariableEnvironmentScopes: args.ExcludeVariableEnvironmentScopes,
		},
		IgnoreCacErrors:      false,
		InlineVariableValues: false,
	}
	libraryVariableSetConverter := converters.LibraryVariableSetConverter{
		Client:               &client,
		VariableSetConverter: &variableSetConverter,
		Excluder:             converters.DefaultExcluder{},
	}

	workerPoolProcessor := converters.OctopusWorkerPoolProcessor{
		WorkerPoolConverter:     workerPoolConverter,
		LookupDefaultWorkerPool: false,
		Client:                  &client,
	}

	runbookConverter := converters.RunbookConverter{
		Client: &client,
		RunbookProcessConverter: converters.RunbookProcessConverter{
			Client: &client,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:          feedConverter,
				AccountConverter:       accountConverter,
				WorkerPoolConverter:    workerPoolConverter,
				EnvironmentConverter:   environmentConverter,
				DetachProjectTemplates: true,
				WorkerPoolProcessor:    workerPoolProcessor,
				StepTemplateConverter:  stepTemplateConverter,
			},
			IgnoreProjectChanges:       false,
			WorkerPoolProcessor:        workerPoolProcessor,
			ExcludeTenantTags:          nil,
			ExcludeTenantTagSets:       nil,
			Excluder:                   converters.DefaultExcluder{},
			TagSetConverter:            &tagsetConverter,
			IgnoreInvalidExcludeExcept: false,
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
		Client:                      &client,
		LifecycleConverter:          lifecycleConverter,
		GitCredentialsConverter:     gitCredentialsConverter,
		LibraryVariableSetConverter: &libraryVariableSetConverter,
		ProjectGroupConverter:       projectGroupConverter,
		DeploymentProcessConverter: converters.DeploymentProcessConverter{
			Client: &client,
			OctopusActionProcessor: converters.OctopusActionProcessor{
				FeedConverter:          feedConverter,
				AccountConverter:       accountConverter,
				WorkerPoolConverter:    workerPoolConverter,
				EnvironmentConverter:   environmentConverter,
				DetachProjectTemplates: true,
				WorkerPoolProcessor:    workerPoolProcessor,
				StepTemplateConverter:  stepTemplateConverter,
			},
			IgnoreProjectChanges:       false,
			WorkerPoolProcessor:        workerPoolProcessor,
			ExcludeTenantTags:          nil,
			ExcludeTenantTagSets:       nil,
			Excluder:                   converters.DefaultExcluder{},
			TagSetConverter:            &tagsetConverter,
			LimitAttributeLength:       0,
			ExcludeTerraformVariables:  false,
			ExcludeAllSteps:            false,
			ExcludeSteps:               nil,
			ExcludeStepsRegex:          nil,
			ExcludeStepsExcept:         nil,
			IgnoreInvalidExcludeExcept: false,
			DummySecretGenerator:       nil,
			DummySecretVariableValues:  false,
			IgnoreCacErrors:            false,
		},
		TenantConverter: &tenantConverter,
		ProjectTriggerConverter: converters.ProjectTriggerConverter{
			Client:               &client,
			EnvironmentConverter: environmentConverter,
		},
		VariableSetConverter:                  &variableSetConverter,
		ChannelConverter:                      channelConverter,
		RunbookConverter:                      &runbookConverter,
		IgnoreCacManagedValues:                false,
		ExcludeCaCProjectSettings:             false,
		ExcludeAllRunbooks:                    false,
		IgnoreProjectChanges:                  false,
		IgnoreProjectGroupChanges:             false,
		IgnoreProjectNameChanges:              false,
		ExcludeProjects:                       nil,
		ExcludeProjectsExcept:                 nil,
		ExcludeProjectsRegex:                  nil,
		ExcludeAllProjects:                    false,
		DummySecretVariableValues:             false,
		DummySecretGenerator:                  nil,
		Excluder:                              converters.DefaultExcluder{},
		LookupOnlyMode:                        false,
		ErrGroup:                              nil,
		ExcludeTerraformVariables:             false,
		IncludeIds:                            false,
		LimitResourceCount:                    0,
		IncludeSpaceInPopulation:              false,
		GenerateImportScripts:                 false,
		TenantCommonVariableProcessor:         tenantCommonVariableProcessor,
		ExportTenantCommonVariablesForProject: true,
		LookupProjectLinkTenants:              false,
		TenantProjectConverter:                tenantProjectConverter,
		TenantVariableConverter:               tenantVariableConverter,
		EnvironmentConverter:                  environmentConverter,
		IgnoreCacErrors:                       false,
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
