package main

import (
	"errors"
	"fmt"
	officialclient "github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	args2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/entry"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/output"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformTestFramework/test"
	"github.com/google/uuid"
	cp "github.com/otiai10/copy"
	"github.com/samber/lo"
	"k8s.io/utils/strings/slices"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// getTempDir creates a temporary directory for the exported Terraform files
func getTempDir() string {
	return os.TempDir() + string(os.PathSeparator) + uuid.New().String() + string(os.PathSeparator)
}

// createClient creates a client used to access the Octopus API
func createClient(container *test.OctopusContainer, space string) client.OctopusClient {
	return &client.OctopusApiClient{
		Url:    container.URI,
		Space:  space,
		ApiKey: test.ApiKey,
	}
}

func copyDir(source string) (string, error) {
	if source == "" {
		return "", nil
	}

	dest, err := os.MkdirTemp("", "octoterra")
	if err != nil {
		return "", err
	}
	err = cp.Copy(source, dest)

	return dest, err
}

// exportSpaceImportAndTest creates a reference space, exports it, and reimports the export
func exportSpaceImportAndTest(
	t *testing.T,
	createSourceBlankSpaceModuleDir string,
	populateSourceSpaceModuleDir string,
	createSourceSpaceVars []string,
	importSpaceVars []string,
	arguments args2.Arguments,
	testFunc func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error) {

	/*
		The directory holding the module to create the space must be copied to allow for parallel
		test execution.
	*/
	createSpaceDirCopy, err := copyDir("../test/terraform/z-createspace")

	if err != nil {
		t.Fatalf(err.Error())
	}

	populateSourceSpaceModuleDirCopy, err := copyDir(populateSourceSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	defer func() {
		for _, dir := range []string{populateSourceSpaceModuleDirCopy, createSpaceDirCopy} {
			err := os.RemoveAll(dir)
			if err != nil {
				t.Fatalf(err.Error())
			}
		}

	}()

	exportImportAndTest(
		t,
		createSourceBlankSpaceModuleDir,
		"",
		populateSourceSpaceModuleDirCopy,
		createSpaceDirCopy,
		"",
		createSourceSpaceVars,
		[]string{},
		[]string{},
		importSpaceVars,
		func(url string, space string, apiKey string, dest string) error {
			args := args2.Arguments{
				Url:                              url,
				ApiKey:                           test.ApiKey,
				Space:                            space,
				Destination:                      dest,
				Console:                          true,
				ProjectId:                        []string{},
				ProjectName:                      []string{},
				LookupProjectDependencies:        false,
				IgnoreCacManagedValues:           arguments.IgnoreCacManagedValues,
				BackendBlock:                     arguments.BackendBlock,
				DetachProjectTemplates:           arguments.DetachProjectTemplates,
				DefaultSecretVariableValues:      arguments.DefaultSecretVariableValues,
				ProviderVersion:                  arguments.ProviderVersion,
				ExcludeAllRunbooks:               arguments.ExcludeAllRunbooks,
				ExcludeRunbooks:                  arguments.ExcludeRunbooks,
				ExcludeRunbooksRegex:             arguments.ExcludeRunbooksRegex,
				ExcludeProvider:                  arguments.ExcludeProvider,
				IncludeOctopusOutputVars:         arguments.IncludeOctopusOutputVars,
				ExcludeLibraryVariableSets:       arguments.ExcludeLibraryVariableSets,
				ExcludeLibraryVariableSetsRegex:  arguments.ExcludeLibraryVariableSetsRegex,
				IgnoreProjectChanges:             arguments.IgnoreProjectChanges,
				IgnoreProjectVariableChanges:     arguments.IgnoreProjectVariableChanges,
				IgnoreProjectGroupChanges:        arguments.IgnoreProjectGroupChanges,
				IgnoreProjectNameChanges:         arguments.IgnoreProjectNameChanges,
				ExcludeProjectVariables:          arguments.ExcludeProjectVariables,
				ExcludeProjectVariablesRegex:     arguments.ExcludeProjectVariablesRegex,
				ExcludeVariableEnvironmentScopes: arguments.ExcludeVariableEnvironmentScopes,
				LookUpDefaultWorkerPools:         arguments.LookUpDefaultWorkerPools,
				ExcludeTenants:                   arguments.ExcludeTenants,
				ExcludeAllTenants:                arguments.ExcludeAllTenants,
				ExcludeProjects:                  arguments.ExcludeProjects,
				ExcludeAllTargets:                arguments.ExcludeAllTargets,
				DummySecretVariableValues:        arguments.DummySecretVariableValues,
				ExcludeAllProjects:               arguments.ExcludeAllProjects,
				ExcludeProjectsRegex:             arguments.ExcludeProjectsRegex,
				ExcludeTenantsExcept:             arguments.ExcludeTenantsExcept,
				ExcludeTenantsWithTags:           arguments.ExcludeTenantsWithTags,
				ExcludeTenantTags:                arguments.ExcludeTenantTags,
				ExcludeTenantTagSets:             arguments.ExcludeTenantTagSets,
				RunbookId:                        arguments.RunbookId,
				RunbookName:                      arguments.RunbookName,
				ExcludeAllProjectVariables:       arguments.ExcludeAllProjectVariables,
				ExcludeTargetsRegex:              arguments.ExcludeTargetsRegex,
				ExcludeTargetsExcept:             arguments.ExcludeTargetsExcept,
				ExcludeProjectsExcept:            arguments.ExcludeProjectsExcept,
				ExcludeTargets:                   arguments.ExcludeTargets,
				ExcludeTenantsRegex:              arguments.ExcludeTenantsRegex,
				ExcludeRunbooksExcept:            arguments.ExcludeRunbooksExcept,
				ExcludeAllLibraryVariableSets:    arguments.ExcludeAllLibraryVariableSets,
				ExcludeLibraryVariableSetsExcept: arguments.ExcludeLibraryVariableSetsExcept,
				ExcludeProjectVariablesExcept:    arguments.ExcludeProjectVariablesExcept,
				ExcludeAllProjectGroups:          arguments.ExcludeAllProjectGroups,
				ExcludeProjectGroups:             arguments.ExcludeProjectGroups,
				ExcludeProjectGroupsRegex:        arguments.ExcludeProjectGroupsRegex,
				ExcludeProjectGroupsExcept:       arguments.ExcludeProjectGroupsExcept,
			}

			dependencies, err := entry.ConvertSpaceToTerraform(args)

			if err != nil {
				return err
			}

			files, err := entry.ProcessResources(dependencies.Resources)

			if err != nil {
				return err
			}

			return output.WriteFiles(strutil.UnEscapeDollarInMap(files), args.Destination, args.Console)
		},
		testFunc)
}

// exportProjectImportAndTest creates a reference space, exports a single project, and reimports the export
func exportProjectImportAndTest(
	t *testing.T,
	projectName string,
	createSourceBlankSpaceModuleDir string,
	populateSourceSpaceModuleDir string,
	createImportBlankSpaceModuleDir string,
	initialiseVars []string,
	initializeSpaceVars []string,
	importSpaceVars []string,
	arguments args2.Arguments,
	testFunc func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error) {

	/*
		The directory holding the module to create the space must be copied to allow for parallel
		test execution.
	*/
	createSourceBlankSpaceModuleDirCopy, err := copyDir(createSourceBlankSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}
	createImportBlankSpaceModuleDirCopy, err := copyDir(createImportBlankSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}

	populateSourceSpaceModuleDirCopy, err := copyDir(populateSourceSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	defer func() {
		for _, dir := range []string{populateSourceSpaceModuleDirCopy, createImportBlankSpaceModuleDirCopy, createSourceBlankSpaceModuleDirCopy} {
			err := os.RemoveAll(dir)
			if err != nil {
				t.Fatalf(err.Error())
			}
		}

	}()

	exportImportAndTest(
		t,
		createSourceBlankSpaceModuleDirCopy,
		"",
		populateSourceSpaceModuleDirCopy,
		createImportBlankSpaceModuleDirCopy,
		"",
		initialiseVars,
		initializeSpaceVars,
		[]string{},
		importSpaceVars,
		func(url string, space string, apiKey string, dest string) error {
			projectId, err := entry.ConvertProjectNameToId(url, space, test.ApiKey, projectName)

			if err != nil {
				return err
			}

			args := args2.Arguments{
				Url:                              url,
				ApiKey:                           test.ApiKey,
				Space:                            space,
				Destination:                      dest,
				Console:                          true,
				ProjectId:                        []string{projectId},
				ProjectName:                      []string{},
				LookupProjectDependencies:        false,
				IgnoreCacManagedValues:           arguments.IgnoreCacManagedValues,
				BackendBlock:                     arguments.BackendBlock,
				DetachProjectTemplates:           arguments.DetachProjectTemplates,
				DefaultSecretVariableValues:      arguments.DefaultSecretVariableValues,
				ProviderVersion:                  arguments.ProviderVersion,
				ExcludeAllRunbooks:               arguments.ExcludeAllRunbooks,
				ExcludeRunbooks:                  arguments.ExcludeRunbooks,
				ExcludeRunbooksRegex:             arguments.ExcludeRunbooksRegex,
				ExcludeProvider:                  arguments.ExcludeProvider,
				IncludeOctopusOutputVars:         arguments.IncludeOctopusOutputVars,
				ExcludeLibraryVariableSets:       arguments.ExcludeLibraryVariableSets,
				ExcludeLibraryVariableSetsRegex:  arguments.ExcludeLibraryVariableSetsRegex,
				IgnoreProjectChanges:             arguments.IgnoreProjectChanges,
				IgnoreProjectVariableChanges:     arguments.IgnoreProjectVariableChanges,
				IgnoreProjectGroupChanges:        arguments.IgnoreProjectGroupChanges,
				IgnoreProjectNameChanges:         arguments.IgnoreProjectNameChanges,
				ExcludeProjectVariables:          arguments.ExcludeProjectVariables,
				ExcludeProjectVariablesRegex:     arguments.ExcludeProjectVariablesRegex,
				ExcludeVariableEnvironmentScopes: arguments.ExcludeVariableEnvironmentScopes,
				LookUpDefaultWorkerPools:         arguments.LookUpDefaultWorkerPools,
				ExcludeTenants:                   arguments.ExcludeTenants,
				ExcludeAllTenants:                arguments.ExcludeAllTenants,
				ExcludeProjects:                  arguments.ExcludeProjects,
				ExcludeAllTargets:                arguments.ExcludeAllTargets,
				DummySecretVariableValues:        arguments.DummySecretVariableValues,
				ExcludeAllProjects:               arguments.ExcludeAllProjects,
				ExcludeProjectsRegex:             arguments.ExcludeProjectsRegex,
				ExcludeTenantsExcept:             arguments.ExcludeTenantsExcept,
				ExcludeTenantsWithTags:           arguments.ExcludeTenantsWithTags,
				ExcludeTenantTags:                arguments.ExcludeTenantTags,
				ExcludeTenantTagSets:             arguments.ExcludeTenantTagSets,
				RunbookId:                        arguments.RunbookId,
				RunbookName:                      arguments.RunbookName,
				ExcludeAllProjectVariables:       arguments.ExcludeAllProjectVariables,
				ExcludeTargetsRegex:              arguments.ExcludeTargetsRegex,
				ExcludeTargetsExcept:             arguments.ExcludeTargetsExcept,
				ExcludeProjectsExcept:            arguments.ExcludeProjectsExcept,
				ExcludeTargets:                   arguments.ExcludeTargets,
				ExcludeTenantsRegex:              arguments.ExcludeTenantsRegex,
				ExcludeRunbooksExcept:            arguments.ExcludeRunbooksExcept,
				ExcludeAllLibraryVariableSets:    arguments.ExcludeAllLibraryVariableSets,
				ExcludeLibraryVariableSetsExcept: arguments.ExcludeLibraryVariableSetsExcept,
				ExcludeProjectVariablesExcept:    arguments.ExcludeProjectVariablesExcept,
			}

			dependencies, err := entry.ConvertProjectToTerraform(args)

			if err != nil {
				return err
			}

			files, err := entry.ProcessResources(dependencies.Resources)

			if err != nil {
				return err
			}

			return output.WriteFiles(strutil.UnEscapeDollarInMap(files), args.Destination, args.Console)
		},
		testFunc)
}

// exportProjectLookupImportAndTest is used to create the initial space with a space prepopulation and a space
// popultion module. The project is then exported. The destination space is created with the same space prepopulation
// module, and the project is imported. This flow means the prepopulation module contains the shared "external"
// resources that an exported project references.
func exportProjectLookupImportAndTest(
	t *testing.T,
	projectName string,
	createSourceBlankSpaceModuleDir string,
	prepopulateSourceBlankSpaceModuleDir string,
	populateSourceSpaceModuleDir string,
	createImportBlankSpaceModuleDir string,
	prepopulateImportSpaceModuleDir string,
	createSourceSpaceVars []string,
	createImportSpaceVars []string,
	prepopulateSpaceVars []string,
	importSpaceVars []string,
	argumnets args2.Arguments,
	testFunc func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error) {

	prepopulateImportSpaceModuleDirCopy, err := copyDir(prepopulateImportSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}
	prepopulateSourceBlankSpaceModuleDirCopy, err := copyDir(prepopulateSourceBlankSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}

	createSourceBlankSpaceModuleDirCopy, err := copyDir(createSourceBlankSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}

	createImportBlankSpaceModuleDirCopy, err := copyDir(createImportBlankSpaceModuleDir)

	if err != nil {
		t.Fatalf(err.Error())
	}

	populateSourceSpaceModuleDirCopy, err := copyDir(populateSourceSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	defer func() {
		for _, dir := range []string{populateSourceSpaceModuleDirCopy, createImportBlankSpaceModuleDirCopy, createSourceBlankSpaceModuleDirCopy, prepopulateSourceBlankSpaceModuleDirCopy, prepopulateImportSpaceModuleDirCopy} {
			err := os.RemoveAll(dir)
			if err != nil {
				t.Fatalf(err.Error())
			}
		}

	}()

	exportImportAndTest(
		t,
		createSourceBlankSpaceModuleDirCopy,
		prepopulateSourceBlankSpaceModuleDirCopy,
		populateSourceSpaceModuleDirCopy,
		createImportBlankSpaceModuleDirCopy,
		prepopulateImportSpaceModuleDirCopy,
		createSourceSpaceVars,
		createImportSpaceVars,
		prepopulateSpaceVars,
		importSpaceVars,
		func(url string, space string, apiKey string, dest string) error {
			projectId, err := entry.ConvertProjectNameToId(url, space, test.ApiKey, projectName)
			if err != nil {
				return err
			}

			runbookId := ""
			if argumnets.RunbookName != "" {
				runbookId, err = entry.ConvertRunbookNameToId(url, space, test.ApiKey, projectId, argumnets.RunbookName)

				if err != nil {
					return err
				}
			}

			args := args2.Arguments{
				Url:                              url,
				ApiKey:                           test.ApiKey,
				Space:                            space,
				Destination:                      dest,
				Console:                          true,
				ProjectId:                        []string{projectId},
				ProjectName:                      []string{},
				LookupProjectDependencies:        true,
				IgnoreCacManagedValues:           argumnets.IgnoreCacManagedValues,
				BackendBlock:                     argumnets.BackendBlock,
				DetachProjectTemplates:           argumnets.DetachProjectTemplates,
				DefaultSecretVariableValues:      argumnets.DefaultSecretVariableValues,
				ProviderVersion:                  argumnets.ProviderVersion,
				ExcludeAllRunbooks:               argumnets.ExcludeAllRunbooks,
				ExcludeRunbooks:                  argumnets.ExcludeRunbooks,
				ExcludeRunbooksRegex:             argumnets.ExcludeRunbooksRegex,
				ExcludeProvider:                  argumnets.ExcludeProvider,
				IncludeOctopusOutputVars:         argumnets.IncludeOctopusOutputVars,
				ExcludeLibraryVariableSets:       argumnets.ExcludeLibraryVariableSets,
				ExcludeLibraryVariableSetsRegex:  argumnets.ExcludeLibraryVariableSetsRegex,
				IgnoreProjectChanges:             argumnets.IgnoreProjectChanges,
				IgnoreProjectVariableChanges:     argumnets.IgnoreProjectVariableChanges,
				IgnoreProjectGroupChanges:        argumnets.IgnoreProjectGroupChanges,
				IgnoreProjectNameChanges:         argumnets.IgnoreProjectNameChanges,
				ExcludeProjectVariables:          argumnets.ExcludeProjectVariables,
				ExcludeProjectVariablesRegex:     argumnets.ExcludeProjectVariablesRegex,
				ExcludeVariableEnvironmentScopes: argumnets.ExcludeVariableEnvironmentScopes,
				LookUpDefaultWorkerPools:         argumnets.LookUpDefaultWorkerPools,
				ExcludeTenants:                   argumnets.ExcludeTenants,
				ExcludeAllTenants:                argumnets.ExcludeAllTenants,
				ExcludeProjects:                  argumnets.ExcludeProjects,
				ExcludeAllTargets:                argumnets.ExcludeAllTargets,
				RunbookId:                        runbookId,
				RunbookName:                      "",
				LookupProjectLinkTenants:         argumnets.LookupProjectLinkTenants,
			}

			var dependencies *data.ResourceDetailsCollection = nil
			if args.RunbookId != "" {
				dependencies, err = entry.ConvertRunbookToTerraform(args)
			} else {
				dependencies, err = entry.ConvertProjectToTerraform(args)
			}

			if err != nil {
				return err
			}

			files, err := entry.ProcessResources(dependencies.Resources)

			if err != nil {
				return err
			}

			return output.WriteFiles(strutil.UnEscapeDollarInMap(files), args.Destination, args.Console)
		},
		testFunc)
}

// exportSpaceImportAndTest imports the sample space, exports the space as Terraform, reimports it as a new space, and executes a callback
// to verify the results.
//
// There is some terminology used in the arguments to make things easier:
// * "source" space is the space populated by TF modules included in this project. This is the space that tests export from.
// * "import" space is the space that the TF modules generated by exporting the source space are applied to.
//
// The process for initialising both the source and import spaces is this:
// 1. A new blank space is created.
// 2. The blank space is optionally populated with some shared global resources. This is only done when testing the export of projects that use data source lookups to reference existing resources. An empty string passed as the prepopulate directory disables this step.
// 3. The source space is populated with the supplied TF files, while the import space is populated with the TF files generated by exporting the source space.
//
// createSourceBlankSpaceModuleDir is the directory that is used to create the source space.
// prepopulateSourceSpaceModuleDir is the optional directory containing a TF module to prepopulate the source space.
// populateSourceSpaceModuleDir is the directory used to populate the new source space.
// createImportBlankSpaceModuleDir is the directory used to create the new import space. This is usually the same as createSourceBlankSpaceModuleDir.
// prepopulateImportSpaceModuleDir is an optional directory that is used to prepopulate the import space. This is usually the same as prepopulateSourceSpaceModuleDir to ensure the source space and import space have the same "global" resources.
// createSourceSpaceVars are the TF vars used when creating the source space.
// createImportSpaceVars are the TF vars used when creating the new import space.
// importSpaceVars are the TF vars used when populating the imported space.
func exportImportAndTest(
	t *testing.T,
	createSourceBlankSpaceModuleDir string,
	prepopulateSourceSpaceModuleDir string,
	populateSourceSpaceModuleDir string,
	createImportBlankSpaceModuleDir string,
	prepopulateImportSpaceModuleDir string,
	createSourceSpaceVars []string,
	createImportSpaceVars []string,
	prePopulateSpaceVars []string,
	importSpaceVars []string,
	exportFunc func(url string, space string, apiKey string, dest string) error,
	testFunc func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error) {

	t.Parallel()

	prepopulateImportSpaceModuleDirCopy, err := copyDir(prepopulateImportSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	createSourceBlankSpaceModuleDirCopy, err := copyDir(createSourceBlankSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	createImportBlankSpaceModuleDirCopy, err := copyDir(createImportBlankSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	populateSourceSpaceModuleDirCopy, err := copyDir(populateSourceSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	prepopulateSourceSpaceModuleDirCopy, err := copyDir(prepopulateSourceSpaceModuleDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	defer func() {
		for _, dir := range []string{populateSourceSpaceModuleDirCopy, createImportBlankSpaceModuleDirCopy, createSourceBlankSpaceModuleDirCopy, prepopulateImportSpaceModuleDirCopy, prepopulateSourceSpaceModuleDirCopy} {
			err := os.RemoveAll(dir)
			if err != nil {
				t.Fatalf(err.Error())
			}
		}

	}()

	testFramework := test.OctopusContainerTest{}
	testFramework.ArrangeTest(t, func(t *testing.T, container *test.OctopusContainer, spaceClient *officialclient.Client) (funcErr error) {
		octopusClient := createClient(container, "")

		// Act
		newSpaceId, err := testFramework.ActWithCustomPrePopulatedSpace(
			t,
			container,
			createSourceBlankSpaceModuleDirCopy,
			prepopulateSourceSpaceModuleDirCopy,
			populateSourceSpaceModuleDirCopy,
			createImportSpaceVars,
			prePopulateSpaceVars,
			createSourceSpaceVars)

		if err != nil {
			return err
		}

		t.Log("EXPORTING TEST SPACE \"" + newSpaceId + "\" (" + container.URI + ")")

		tempDir := getTempDir()
		defer func(name string) {
			err := os.RemoveAll(name)
			if err != nil {
				funcErr = errors.Join(funcErr, err)
			}
		}(tempDir)

		err = exportFunc(container.URI, newSpaceId, test.ApiKey, tempDir)

		if err != nil {
			return err
		}

		t.Log("REIMPORTING TEST SPACE (" + container.URI + ")")

		populateSpaceDir := filepath.Join(tempDir, "space_population")

		err = testFramework.InitialiseOctopus(
			t,
			container,
			createImportBlankSpaceModuleDirCopy,
			prepopulateImportSpaceModuleDirCopy,
			populateSpaceDir,
			"Test3",
			createImportSpaceVars,
			prePopulateSpaceVars,
			importSpaceVars)

		if err != nil {
			// There are some odd errors where Terraform thinks "Test3" is an existing space.
			// So dump the existing spaces if we get an error just to confirm.
			spaces, spacesErr := octopusClient.GetSpaces()

			t.Log("Existing spaces")
			if spacesErr == nil {
				for _, space := range spaces {
					t.Log("ID: " + space.Id + " Name: " + space.Name + " Description: " + strutil.EmptyIfNil(space.Description))
				}
			} else {
				t.Log(spacesErr.Error())
			}

			return err
		}

		recreatedSpaceId, err := testFramework.GetOutputVariable(t, createImportBlankSpaceModuleDirCopy, "octopus_space_id")

		if err != nil || len(strings.TrimSpace(recreatedSpaceId)) == 0 {
			/*
					There is an intermittent bug where the state saved by Terraform is this empty JSON, despite
					everything appearing to work as expected:
				    {"format_version":"1.0"}

				    In the event of an error, assume the new space is Spaces-3.
			*/
			t.Log("Falling back to default value of Spaces-3")
			recreatedSpaceId = "Spaces-3"
		}

		err = testFunc(t, container, recreatedSpaceId, populateSpaceDir)

		return err
	})
}

// TestSpaceExport verifies that a space can be reimported with the correct settings
func TestSpaceExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/1-singlespace/space_creation",
		"../test/terraform/1-singlespace/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			space := octopus.Space{}
			err := octopusClient.GetSpace(&space)

			if err != nil {
				return err
			}

			if space.Name != "Test3" {
				return errors.New("New space must have the name \"Test3\"")
			}

			// The description is overridden by the test framework to reflect the name of the test
			if *space.Description != "TestSpaceExport" {
				return errors.New("New space must have the name \"TestSpaceExport\"")
			}

			if space.IsDefault {
				return errors.New("New space must not be the default one")
			}

			if space.TaskQueueStopped {
				return errors.New("New space must not have the task queue stopped")
			}

			if slices.Index(space.SpaceManagersTeams, "teams-administrators") == -1 {
				return errors.New("New space must have teams-administrators as a manager team")
			}

			return nil
		})
}

// TestProjectGroupExport verifies that a project group can be reimported with the correct settings
func TestProjectGroupExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/2-projectgroup/space_creation",
		"../test/terraform/2-projectgroup/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeProjectGroupsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
			err := octopusClient.GetAllResources("ProjectGroups", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "Test" {
					found = true
					if strutil.EmptyIfNil(v.Description) != "Test Description" {
						return errors.New("The project group must be have a description of \"Test Description\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have a project group called \"Test\"")
			}

			return nil
		})
}

// TestAwsAccountExport verifies that an AWS account can be reimported with the correct settings
func TestAwsAccountExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/3-awsaccount/space_creation",
		"../test/terraform/3-awsaccount/space_population",
		[]string{},
		[]string{"-var=account_aws_account=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "AWS Account" {
					found = true
					if strutil.EmptyIfNil(v.AccessKey) != "ABCDEFGHIJKLMNOPQRST" {
						return errors.New("The account must be have an access key of \"ABCDEFGHIJKLMNOPQRST\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"AWS Account\"")
			}

			return nil
		})
}

// TestAzureAccountExport verifies that an Azure account can be reimported with the correct settings
func TestAzureAccountExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/4-azureaccount/space_creation",
		"../test/terraform/4-azureaccount/space_population",
		[]string{},
		[]string{"-var=account_azure=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "Azure" {
					found = true
					if strutil.EmptyIfNil(v.ClientId) != "2eb8bd13-661e-489c-beb9-4103efb9dbdd" {
						return errors.New("The account must be have a client ID of \"2eb8bd13-661e-489c-beb9-4103efb9dbdd\"")
					}

					if strutil.EmptyIfNil(v.SubscriptionNumber) != "95bf77d2-64b1-4ed2-9de1-b5451e3881f5" {
						return errors.New("The account must be have a client ID of \"95bf77d2-64b1-4ed2-9de1-b5451e3881f5\"")
					}

					if strutil.EmptyIfNil(v.TenantId) != "18eb006b-c3c8-4a72-93cd-fe4b293f82ee" {
						return errors.New("The account must be have a client ID of \"18eb006b-c3c8-4a72-93cd-fe4b293f82ee\"")
					}

					if strutil.EmptyIfNil(v.Description) != "Azure Account" {
						return errors.New("The account must be have a description of \"Azure Account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Tenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Tenanted\"")
					}

					if len(v.EnvironmentIds) != 1 {
						return errors.New("The account must be scoped to an environment")
					}

					if len(v.TenantIds) != 1 {
						return errors.New("The account must be scoped to an tenant")
					}
				}
			}

			if !found {
				return errors.New("Space must have an Azure account called \"Azure\"")
			}

			return nil
		})
}

// TestUsernamePasswordAccountExport verifies that a username/password account can be reimported with the correct settings
func TestUsernamePasswordAccountExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/5-userpassaccount/space_creation",
		"../test/terraform/5-userpassaccount/space_population",
		[]string{},
		[]string{"-var=account_gke=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "GKE" {
					found = true
					if strutil.EmptyIfNil(v.Username) != "admin" {
						return errors.New("The account must be have a username of \"admin\"")
					}

					if !v.Password.HasValue {
						return errors.New("The account must be have a password")
					}

					if strutil.EmptyIfNil(v.Description) != "A test account" {
						return errors.New("The account must be have a description of \"A test account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Untenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Untenanted\"")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The account must be have no tenant tags")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"GKE\"")
			}

			return nil
		})
}

// TestGcpAccountExport verifies that a GCP account can be reimported with the correct settings
func TestGcpAccountExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/6-gcpaccount/space_creation",
		"../test/terraform/6-gcpaccount/space_population",
		[]string{},
		[]string{"-var=account_google=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			found := false
			for _, v := range collection.Items {
				if v.Name == "Google" {
					found = true
					if !v.JsonKey.HasValue {
						return errors.New("The account must be have a JSON key")
					}

					if strutil.EmptyIfNil(v.Description) != "A test account" {
						return errors.New("The account must be have a description of \"A test account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Untenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Untenanted\"")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The account must be have no tenant tags")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"Google\"")
			}

			return nil
		})
}

// TestSshAccountExport verifies that a SSH account can be reimported with the correct settings
func TestSshAccountExport(t *testing.T) {
	// We set the passphrase because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
	exportSpaceImportAndTest(
		t,
		"../test/terraform/7-sshaccount/space_creation",
		"../test/terraform/7-sshaccount/space_population",
		[]string{},
		[]string{
			"-var=account_ssh_cert=whatever",
			"-var=account_ssh=LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFRRUF5c25PVXhjN0tJK2pIRUc5RVEwQXFCMllGRWE5ZnpZakZOY1pqY1dwcjJQRkRza25oOUpTCm1NVjVuZ2VrbTRyNHJVQU5tU2dQMW1ZTGo5TFR0NUVZa0N3OUdyQ0paNitlQTkzTEowbEZUamFkWEJuQnNmbmZGTlFWYkcKZ2p3U1o4SWdWQ2oySXE0S1hGZm0vbG1ycEZQK2Jqa2V4dUxwcEh5dko2ZmxZVjZFMG13YVlneVNHTWdLYy9ubXJaMTY0WApKMStJL1M5NkwzRWdOT0hNZmo4QjM5eEhZQ0ZUTzZEQ0pLQ3B0ZUdRa0gwTURHam84d3VoUlF6c0IzVExsdXN6ZG0xNmRZCk16WXZBSWR3emZ3bzh1ajFBSFFOendDYkIwRmR6bnFNOEpLV2ZrQzdFeVVrZUl4UXZmLzJGd1ZyS0xEZC95ak5PUmNoa3EKb2owNncySXFad0FBQThpS0tqT3dpaW96c0FBQUFBZHpjMmd0Y25OaEFBQUJBUURLeWM1VEZ6c29qNk1jUWIwUkRRQ29IWgpnVVJyMS9OaU1VMXhtTnhhbXZZOFVPeVNlSDBsS1l4WG1lQjZTYml2aXRRQTJaS0EvV1pndVAwdE8za1JpUUxEMGFzSWxuCnI1NEQzY3NuU1VWT05wMWNHY0d4K2Q4VTFCVnNhQ1BCSm53aUJVS1BZaXJncGNWK2IrV2F1a1UvNXVPUjdHNHVta2ZLOG4KcCtWaFhvVFNiQnBpREpJWXlBcHorZWF0blhyaGNuWDRqOUwzb3ZjU0EwNGN4K1B3SGYzRWRnSVZNN29NSWtvS20xNFpDUQpmUXdNYU9qekM2RkZET3dIZE11VzZ6TjJiWHAxZ3pOaThBaDNETi9Dank2UFVBZEEzUEFKc0hRVjNPZW96d2twWitRTHNUCkpTUjRqRkM5Ly9ZWEJXc29zTjMvS00wNUZ5R1NxaVBUckRZaXBuQUFBQUF3RUFBUUFBQVFFQXdRZzRqbitlb0kyYUJsdk4KVFYzRE1rUjViMU9uTG1DcUpEeGM1c2N4THZNWnNXbHBaN0NkVHk4ckJYTGhEZTdMcUo5QVVub0FHV1lwdTA1RW1vaFRpVwptVEFNVHJCdmYwd2xsdCtJZVdvVXo3bmFBbThQT1psb29MbXBYRzh5VmZKRU05aUo4NWtYNDY4SkF6VDRYZ1JXUFRYQ1JpCi9abCtuWUVUZVE4WTYzWlJhTVE3SUNmK2FRRWxRenBYb21idkxYM1RaNmNzTHh5Z3Eza01aSXNJU0lUcEk3Y0tsQVJ0Rm4KcWxKRitCL2JlUEJkZ3hIRVpqZDhDV0NIR1ZRUDh3Z3B0d0Rrak9NTzh2b2N4YVpOT0hZZnBwSlBCTkVjMEVKbmduN1BXSgorMVZSTWZKUW5SemVubmE3VHdSUSsrclZmdkVaRmhqamdSUk85RitrMUZvSWdRQUFBSUVBbFFybXRiV2V0d3RlWlZLLys4CklCUDZkcy9MSWtPb3pXRS9Wckx6cElBeHEvV1lFTW1QK24wK1dXdWRHNWpPaTFlZEJSYVFnU0owdTRxcE5JMXFGYTRISFYKY2oxL3pzenZ4RUtSRElhQkJGaU81Y3QvRVQvUTdwanozTnJaZVdtK0dlUUJKQ0diTEhSTlQ0M1ZpWVlLVG82ZGlGVTJteApHWENlLzFRY2NqNjVZQUFBQ0JBUHZodmgzb2Q1MmY4SFVWWGoxeDNlL1ZFenJPeVloTi9UQzNMbWhHYnRtdHZ0L0J2SUhxCndxWFpTT0lWWkZiRnVKSCtORHNWZFFIN29yUW1VcGJxRllDd0IxNUZNRGw0NVhLRm0xYjFyS1c1emVQK3d0M1hyM1p0cWsKRkdlaUlRMklSZklBQjZneElvNTZGemdMUmx6QnB0bzhkTlhjMXhtWVgyU2Rhb3ZwSkRBQUFBZ1FET0dwVE9oOEFRMFoxUwpzUm9vVS9YRTRkYWtrSU5vMDdHNGI3M01maG9xbkV1T01LM0ZRVStRRWUwYWpvdWs5UU1QNWJzZU1CYnJNZVNNUjBRWVBCClQ4Z0Z2S2VISWN6ZUtJTjNPRkRaRUF4TEZNMG9LbjR2bmdHTUFtTXUva2QwNm1PZnJUNDRmUUh1ajdGNWx1QVJHejRwYUwKLzRCTUVkMnFTRnFBYzZ6L0RRQUFBQTF0WVhSMGFFQk5ZWFIwYUdWM0FRSURCQT09Ci0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			accountName := "SSH"
			found := false
			for _, v := range collection.Items {
				if v.Name == accountName {
					found = true
					if v.AccountType != "SshKeyPair" {
						return errors.New("The account must be have a type of \"SshKeyPair\"")
					}

					if strutil.EmptyIfNil(v.Username) != "admin" {
						return errors.New("The account must be have a username of \"admin\"")
					}

					if strutil.EmptyIfNil(v.Description) != "A test account" {
						// This appears to be a bug in the provider where the description is not set
						t.Log("BUG: The account must be have a description of \"A test account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Untenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Untenanted\"")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The account must be have no tenant tags")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"" + accountName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureSubscriptionAccountExport verifies that an azure account can be reimported with the correct settings
func TestAzureSubscriptionAccountExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/8-azuresubscriptionaccount/space_creation",
		"../test/terraform/8-azuresubscriptionaccount/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=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",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			accountName := "Subscription"
			found := false
			for _, v := range collection.Items {
				if v.Name == accountName {
					found = true
					if v.AccountType != "AzureSubscription" {
						return errors.New("The account must be have a type of \"AzureSubscription\"")
					}

					if strutil.EmptyIfNil(v.Description) != "A test account" {
						return errors.New("BUG: The account must be have a description of \"A test account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Untenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Untenanted\"")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The account must be have no tenant tags")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"" + accountName + "\"")
			}

			return nil
		})
}

// TestTokenAccountExport verifies that a token account can be reimported with the correct settings
func TestTokenAccountExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/9-tokenaccount/space_creation",
		"../test/terraform/9-tokenaccount/space_population",
		[]string{},
		[]string{
			"-var=account_token=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Account]{}
			err := octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			accountName := "Token"
			found := false
			for _, v := range collection.Items {
				if v.Name == accountName {
					found = true
					if v.AccountType != "Token" {
						return errors.New("The account must be have a type of \"Token\"")
					}

					if !v.Token.HasValue {
						return errors.New("The account must be have a token")
					}

					if strutil.EmptyIfNil(v.Description) != "A test account" {
						return errors.New("The account must be have a description of \"A test account\"")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentParticipation) != "Untenanted" {
						return errors.New("The account must be have a tenanted deployment participation of \"Untenanted\"")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The account must be have no tenant tags")
					}
				}
			}

			if !found {
				return errors.New("Space must have an account called \"" + accountName + "\"")
			}

			return nil
		})
}

// TestHelmFeedExport verifies that a helm feed can be reimported with the correct settings
func TestHelmFeedExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/10-helmfeed/space_creation",
		"../test/terraform/10-helmfeed/space_population",
		[]string{},
		[]string{
			"-var=feed_helm_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Helm"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "Helm" {
						return errors.New("The feed must have a type of \"Helm\"")
					}

					if strutil.EmptyIfNil(v.Username) != "username" {
						return errors.New("The feed must have a username of \"username\"")
					}

					if !v.Password.HasValue {
						return errors.New("The feed must have a password")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://charts.helm.sh/stable/" {
						return errors.New("The feed must be have a URI of \"https://charts.helm.sh/stable/\"")
					}

					foundExecutionTarget := false
					foundNotAcquired := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "NotAcquired" {
							foundNotAcquired = true
						}
					}

					if !(foundExecutionTarget && foundNotAcquired) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestDockerFeedExport verifies that a docker feed can be reimported with the correct settings
func TestDockerFeedExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/11-dockerfeed/space_creation",
		"../test/terraform/11-dockerfeed/space_population",
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Docker"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "Docker" {
						return errors.New("The feed must have a type of \"Docker\"")
					}

					if strutil.EmptyIfNil(v.Username) != "username" {
						return errors.New("The feed must have a username of \"username\"")
					}

					if !v.Password.HasValue {
						return errors.New("The feed must have a password")
					}

					if strutil.EmptyIfNil(v.ApiVersion) != "v1" {
						return errors.New("The feed must be have a API version of \"v1\"")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://index.docker.io" {
						return errors.New("The feed must be have a feed uri of \"https://index.docker.io\"")
					}

					foundExecutionTarget := false
					foundNotAcquired := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "NotAcquired" {
							foundNotAcquired = true
						}
					}

					if !(foundExecutionTarget && foundNotAcquired) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestDockerFeedNoCredsExport verifies that a docker feed with no credentials can be reimported with the correct settings
func TestDockerFeedNoCredsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/65-dockerfeednocreds/space_creation",
		"../test/terraform/65-dockerfeednocreds/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Docker"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "Docker" {
						return errors.New("The feed must have a type of \"Docker\"")
					}

					if strutil.EmptyIfNil(v.Username) != "" {
						return errors.New("The feed must have an empty username")
					}

					if v.Password.HasValue {
						return errors.New("The feed must not have a password")
					}

					if strutil.EmptyIfNil(v.ApiVersion) != "v1" {
						return errors.New("The feed must be have a API version of \"v1\"")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://index.docker.io" {
						return errors.New("The feed must be have a feed uri of \"https://index.docker.io\"")
					}

					foundExecutionTarget := false
					foundNotAcquired := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "NotAcquired" {
							foundNotAcquired = true
						}
					}

					if !(foundExecutionTarget && foundNotAcquired) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestDummyCredsExport verifies that a docker feed with dummy credentials can be reimported with the correct settings.
// Note that there are no variables defined to supply sensitive values during the reapply step. This validates that
// sensitive values have defaults.
func TestDummyCredsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/68-dummycreds/space_creation",
		"../test/terraform/68-dummycreds/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			DummySecretVariableValues: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			err := func() error {
				collection := octopus.GeneralCollection[octopus.Feed]{}
				err := octopusClient.GetAllResources("Feeds", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.Feed, index int) bool {
					return item.Name == "Docker"
				})) != 1 {
					return errors.New("Space must have an feed called \"Docker\" in space " + recreatedSpaceId)
				}

				return nil
			}()

			if err != nil {
				return err
			}

			err = func() error {
				collection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
					return item.Name == "Test"
				})) != 1 {
					return errors.New("Space must have a project called \"Test\" in space " + recreatedSpaceId)
				}

				return nil
			}()

			if err != nil {
				return err
			}

			err = func() error {
				collection := octopus.GeneralCollection[octopus.GitCredentials]{}
				err := octopusClient.GetAllResources("Git-Credentials", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.GitCredentials, index int) bool {
					return item.Name == "test"
				})) != 1 {
					return errors.New("Space must have git credentials called \"test\" in space " + recreatedSpaceId)
				}

				return nil
			}()

			if err != nil {
				return err
			}

			err = func() error {
				collection := octopus.GeneralCollection[octopus.Account]{}
				err := octopusClient.GetAllResources("Accounts", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "AWS Account"
				})) != 1 {
					return errors.New("Space must have an account called \"AWS Account\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "Azure"
				})) != 1 {
					return errors.New("Space must have an account called \"Azure\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "Subscription"
				})) != 1 {
					return errors.New("Space must have an account called \"Subscription\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "Google"
				})) != 1 {
					return errors.New("Space must have an account called \"Google\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "SSH"
				})) != 1 {
					return errors.New("Space must have an account called \"SSH\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "Token"
				})) != 1 {
					return errors.New("Space must have an account called \"Token\" in space " + recreatedSpaceId)
				}

				if len(lo.Filter(collection.Items, func(item octopus.Account, index int) bool {
					return item.Name == "UserPass"
				})) != 1 {
					return errors.New("Space must have an account called \"UserPass\" in space " + recreatedSpaceId)
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestEcrFeedExport verifies that a ecr feed can be reimported with the correct settings
func TestEcrFeedExport(t *testing.T) {

	if os.Getenv("ECR_ACCESS_KEY") == "" {
		t.Fatalf("the ECR_ACCESS_KEY environment variable must be set a valid AWS access key")
	}

	if os.Getenv("ECR_SECRET_KEY") == "" {
		t.Fatalf("the ECR_SECRET_KEY environment variable must be set a valid AWS secret key")
	}

	exportSpaceImportAndTest(
		t,
		"../test/terraform/12-ecrfeed/space_creation",
		"../test/terraform/12-ecrfeed/space_population",
		[]string{
			"-var=feed_ecr_access_key=" + os.Getenv("ECR_ACCESS_KEY"),
			"-var=feed_ecr_secret_key=" + os.Getenv("ECR_SECRET_KEY"),
		},
		[]string{
			"-var=feed_ecr_password=" + os.Getenv("ECR_SECRET_KEY"),
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "ECR"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "AwsElasticContainerRegistry" {
						return errors.New("The feed must have a type of \"AwsElasticContainerRegistry\" (was \"" + strutil.EmptyIfNil(v.FeedType) + "\"")
					}

					if strutil.EmptyIfNil(v.AccessKey) != os.Getenv("ECR_ACCESS_KEY") {
						return errors.New("The feed must have a access key of \"" + os.Getenv("ECR_ACCESS_KEY") + "\" (was \"" + strutil.EmptyIfNil(v.AccessKey) + "\"")
					}

					if !v.SecretKey.HasValue {
						return errors.New("The feed must have a secret key")
					}

					if strutil.EmptyIfNil(v.Region) != "us-east-1" {
						return errors.New("The feed must have a region of \"us-east-1\" (was \"" + strutil.EmptyIfNil(v.Region) + "\"")
					}

					foundExecutionTarget := false
					foundNotAcquired := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "NotAcquired" {
							foundNotAcquired = true
						}
					}

					if !(foundExecutionTarget && foundNotAcquired) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"NotAcquired\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestMavenFeedExport verifies that a maven feed can be reimported with the correct settings
func TestMavenFeedExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/13-mavenfeed/space_creation",
		"../test/terraform/13-mavenfeed/space_population",
		[]string{},
		[]string{
			"-var=feed_maven_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Maven"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "Maven" {
						return errors.New("The feed must have a type of \"Maven\"")
					}

					if strutil.EmptyIfNil(v.Username) != "username" {
						return errors.New("The feed must have a username of \"username\"")
					}

					if !v.Password.HasValue {
						return errors.New("The feed must have a password")
					}

					if intutil.ZeroIfNil(v.DownloadAttempts) != 5 {
						return errors.New("The feed must be have a downloads attempts set to \"5\"")
					}

					if intutil.ZeroIfNil(v.DownloadRetryBackoffSeconds) != 10 {
						return errors.New("The feed must be have a downloads retry backoff set to \"10\"")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://repo.maven.apache.org/maven2/" {
						return errors.New("The feed must be have a feed uri of \"https://repo.maven.apache.org/maven2/\"")
					}

					foundExecutionTarget := false
					foundServer := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "Server" {
							foundServer = true
						}
					}

					if !(foundExecutionTarget && foundServer) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"Server\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestNugetFeedExport verifies that a nuget feed can be reimported with the correct settings
func TestNugetFeedExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/14-nugetfeed/space_creation",
		"../test/terraform/14-nugetfeed/space_population",
		[]string{},
		[]string{
			"-var=feed_nuget_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Nuget"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "NuGet" {
						return errors.New("The feed must have a type of \"NuGet\"")
					}

					if !v.EnhancedMode {
						return errors.New("The feed must have enhanced mode set to true")
					}

					if strutil.EmptyIfNil(v.Username) != "username" {
						return errors.New("The feed must have a username of \"username\"")
					}

					if !v.Password.HasValue {
						return errors.New("The feed must have a password")
					}

					if intutil.ZeroIfNil(v.DownloadAttempts) != 5 {
						return errors.New("The feed must be have a downloads attempts set to \"5\"")
					}

					if intutil.ZeroIfNil(v.DownloadRetryBackoffSeconds) != 10 {
						return errors.New("The feed must be have a downloads retry backoff set to \"10\"")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://index.docker.io" {
						return errors.New("The feed must be have a feed uri of \"https://index.docker.io\"")
					}

					foundExecutionTarget := false
					foundServer := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "Server" {
							foundServer = true
						}
					}

					if !(foundExecutionTarget && foundServer) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"Server\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestWorkerPoolExport verifies that a static worker pool can be reimported with the correct settings
func TestWorkerPoolExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/15-workerpool/space_creation",
		"../test/terraform/15-workerpool/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.WorkerPool]{}
			err := octopusClient.GetAllResources("WorkerPools", &collection)

			if err != nil {
				return err
			}

			workerPoolName := "Docker"
			found := false
			for _, v := range collection.Items {
				if v.Name == workerPoolName {
					found = true

					if v.WorkerPoolType != "StaticWorkerPool" {
						return errors.New("The worker pool must be have a type of \"StaticWorkerPool\" (was \"" + v.WorkerPoolType + "\"")
					}

					if strutil.EmptyIfNil(v.Description) != "A test worker pool" {
						return errors.New("The worker pool must be have a description of \"A test worker pool\" (was \"" + strutil.EmptyIfNil(v.Description) + "\"")
					}

					if v.SortOrder != 3 {
						return errors.New("The worker pool must be have a sort order of \"3\" (was \"" + fmt.Sprint(v.SortOrder) + "\"")
					}

					if v.IsDefault {
						return errors.New("The worker pool must be must not be the default")
					}
				}
			}

			if !found {
				return errors.New("Space must have an worker pool called \"" + workerPoolName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestEnvironmentExport verifies that an environment can be reimported with the correct settings
func TestEnvironmentExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/16-environment/space_creation",
		"../test/terraform/16-environment/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Environment]{}
			err := octopusClient.GetAllResources("Environments", &collection)

			if err != nil {
				return err
			}

			resourceName := "Development"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "A test environment" {
						return errors.New("The environment must be have a description of \"A test environment\" (was \"" + strutil.EmptyIfNil(v.Description) + "\"")
					}

					if !v.AllowDynamicInfrastructure {
						return errors.New("The environment must have dynamic infrastructure enabled.")
					}

					if v.UseGuidedFailure {
						return errors.New("The environment must not have guided failure enabled.")
					}

					jiraIntegration := lo.Filter(v.ExtensionSettings, func(item octopus.Extension, index int) bool {
						return item.ExtensionId == "jira-integration"
					})

					if len(jiraIntegration) != 1 {
						return errors.New("The environment must have Jira integration settings.")
					}

					if jiraEnvironment, ok := jiraIntegration[0].Values["JiraEnvironmentType"]; !ok || jiraEnvironment != "unmapped" {
						return errors.New("The environment must have Jira environment type pf \"unmapped\".")
					}

					jsmIntegration := lo.Filter(v.ExtensionSettings, func(item octopus.Extension, index int) bool {
						return item.ExtensionId == "jiraservicemanagement-integration"
					})

					if len(jsmIntegration) != 1 {
						return errors.New("The environment must have JSM integration settings.")
					}

					if jsmChangeControlled, ok := jsmIntegration[0].Values["JsmChangeControlled"]; !ok || !jsmChangeControlled.(bool) {
						return errors.New("The environment must have jsm integration enabled.")
					}

					serviceNowIntegration := lo.Filter(v.ExtensionSettings, func(item octopus.Extension, index int) bool {
						return item.ExtensionId == "servicenow-integration"
					})

					if len(serviceNowIntegration) != 1 {
						return errors.New("The environment must have service integration settings.")
					}

					if snowChangeControlled, ok := serviceNowIntegration[0].Values["ServiceNowChangeControlled"]; !ok || !snowChangeControlled.(bool) {
						return errors.New("The environment must have service now integration enabled.")
					}
				}
			}

			if !found {
				return errors.New("Space must have an environment called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestLifecycleExport verifies that a lifecycle can be reimported with the correct settings
func TestLifecycleExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/17-lifecycle/space_creation",
		"../test/terraform/17-lifecycle/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Lifecycle]{}
			err := octopusClient.GetAllResources("Lifecycles", &collection)

			if err != nil {
				return err
			}

			resourceName := "Simple"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "A test lifecycle" {
						return errors.New("The lifecycle must be have a description of \"A test lifecycle\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if v.TentacleRetentionPolicy.QuantityToKeep != 30 {
						return errors.New("The lifecycle must be have a tentacle retention policy of \"30\" (was \"" + fmt.Sprint(v.TentacleRetentionPolicy.QuantityToKeep) + "\")")
					}

					if v.TentacleRetentionPolicy.ShouldKeepForever {
						return errors.New("The lifecycle must be have a tentacle retention not set to keep forever")
					}

					if v.TentacleRetentionPolicy.Unit != "Items" {
						return errors.New("The lifecycle must be have a tentacle retention unit set to \"Items\" (was \"" + v.TentacleRetentionPolicy.Unit + "\")")
					}

					if v.ReleaseRetentionPolicy.QuantityToKeep != 1 {
						return errors.New("The lifecycle must be have a release retention policy of \"1\" (was \"" + fmt.Sprint(v.ReleaseRetentionPolicy.QuantityToKeep) + "\")")
					}

					if !v.ReleaseRetentionPolicy.ShouldKeepForever {
						t.Log("BUG: The lifecycle must be have a release retention set to keep forever (known bug - the provider creates this field as false)")
					}

					if v.ReleaseRetentionPolicy.Unit != "Days" {
						return errors.New("The lifecycle must be have a release retention unit set to \"Days\" (was \"" + v.ReleaseRetentionPolicy.Unit + "\")")
					}
				}
			}

			if !found {
				return errors.New("Space must have an lifecycle called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestVariableSetExport verifies that a variable set can be reimported with the correct settings
func TestVariableSetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/18-variableset/space_creation",
		"../test/terraform/18-variableset/space_population",
		[]string{},
		[]string{
			"-var=variables_test_test_secretvariable_1=blah",
		},
		args2.Arguments{
			ExcludeLibraryVariableSets:      []string{"Test2"},
			ExcludeLibraryVariableSetsRegex: []string{"^Test3$"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
			err := octopusClient.GetAllResources("LibraryVariableSets", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 1 {
				return errors.New("Only 1 library variable set must be reimported, as the others are excluded.")
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test variable set" {
						return errors.New("The library variable set must be have a description of \"Test variable set\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					resource := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", v.VariableSetId, &resource)

					firstVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.Variable" })
					secondVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.SecretVariable" })
					thirdVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.TagScopedVariable" })

					if len(firstVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.Variable\"")
					}

					if len(secondVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.SecretVariable\"")
					}

					if len(thirdVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.TagScopedVariable\"")
					}

					if firstVar[0].Type != "String" {
						return errors.New("The library variable set variable must have a type of \"String\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Description) != "Test variable" {
						return errors.New("The library variable set variable must have a description of \"Test variable\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Value) != "True" {
						return errors.New("The library variable set variable must have a value of \"True\"")
					}

					if firstVar[0].IsSensitive {
						return errors.New("The library variable set variable must not be sensitive")
					}

					if !firstVar[0].IsEditable {
						return errors.New("The library variable set variable must be editable")
					}

					if strutil.EmptyIfNil(firstVar[0].Prompt.Description) != "test description" {
						return errors.New("The library variable set variable must have a prompt description of \"test description\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Prompt.Label) != "test label" {
						return errors.New("The library variable set variable must have a prompt label of \"test label\"")
					}

					if !firstVar[0].Prompt.Required {
						return errors.New("The library variable set variable must have a required prompt")
					}

					if firstVar[0].Prompt.DisplaySettings["Octopus.ControlType"] != "Select" {
						return errors.New("The library variable set variable must have a prompt control type of \"Select\"")
					}

					if firstVar[0].Prompt.DisplaySettings["Octopus.SelectOptions"] != "hi|there" {
						return errors.New("The library variable set variable must have a prompt select option of \"hi|there\"")
					}

					if !secondVar[0].IsSensitive {
						return errors.New("The library variable set variable \"Test.SecretVariable\" must be sensitive")
					}

					if len(thirdVar[0].Scope.TenantTag) != 1 {
						return errors.New("The library variable set variable \"Test.TagScopedVariable\" must have tenant tag scopes")
					}

					if thirdVar[0].Scope.TenantTag[0] != "tag1/a" {
						return errors.New("The library variable set variable \"Test.TagScopedVariable\" must have tenant tag scope of \"tag1/a\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an library variable set called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestVariableSetExcludeExceptExport verifies that a variable set can be reimported with the correct settings
func TestVariableSetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/18-variableset/space_creation",
		"../test/terraform/18-variableset/space_population",
		[]string{},
		[]string{
			"-var=variables_test_test_secretvariable_1=blah",
		},
		args2.Arguments{
			ExcludeLibraryVariableSetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
			err := octopusClient.GetAllResources("LibraryVariableSets", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 1 {
				return errors.New("Only 1 library variable set must be reimported, as the others are excluded.")
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test variable set" {
						return errors.New("The library variable set must be have a description of \"Test variable set\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					resource := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", v.VariableSetId, &resource)

					firstVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.Variable" })
					secondVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.SecretVariable" })
					thirdVar := lo.Filter(resource.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Test.TagScopedVariable" })

					if len(firstVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.Variable\"")
					}

					if len(secondVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.SecretVariable\"")
					}

					if len(thirdVar) != 1 {
						return errors.New("The library variable set variable must have a name of \"Test.TagScopedVariable\"")
					}

					if firstVar[0].Type != "String" {
						return errors.New("The library variable set variable must have a type of \"String\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Description) != "Test variable" {
						return errors.New("The library variable set variable must have a description of \"Test variable\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Value) != "True" {
						return errors.New("The library variable set variable must have a value of \"True\"")
					}

					if firstVar[0].IsSensitive {
						return errors.New("The library variable set variable must not be sensitive")
					}

					if !firstVar[0].IsEditable {
						return errors.New("The library variable set variable must be editable")
					}

					if strutil.EmptyIfNil(firstVar[0].Prompt.Description) != "test description" {
						return errors.New("The library variable set variable must have a prompt description of \"test description\"")
					}

					if strutil.EmptyIfNil(firstVar[0].Prompt.Label) != "test label" {
						return errors.New("The library variable set variable must have a prompt label of \"test label\"")
					}

					if !firstVar[0].Prompt.Required {
						return errors.New("The library variable set variable must have a required prompt")
					}

					if firstVar[0].Prompt.DisplaySettings["Octopus.ControlType"] != "Select" {
						return errors.New("The library variable set variable must have a prompt control type of \"Select\"")
					}

					if firstVar[0].Prompt.DisplaySettings["Octopus.SelectOptions"] != "hi|there" {
						return errors.New("The library variable set variable must have a prompt select option of \"hi|there\"")
					}

					if !secondVar[0].IsSensitive {
						return errors.New("The library variable set variable \"Test.SecretVariable\" must be sensitive")
					}

					if len(thirdVar[0].Scope.TenantTag) != 1 {
						return errors.New("The library variable set variable \"Test.TagScopedVariable\" must have tenant tag scopes")
					}

					if thirdVar[0].Scope.TenantTag[0] != "tag1/a" {
						return errors.New("The library variable set variable \"Test.TagScopedVariable\" must have tenant tag scope of \"tag1/a\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an library variable set called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestVariableSetExcludeAllExport verifies that all variable sets can be excluded
func TestVariableSetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/18-variableset/space_creation",
		"../test/terraform/18-variableset/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllLibraryVariableSets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
			err := octopusClient.GetAllResources("LibraryVariableSets", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("All library variable sets must be excluded.")
			}

			return nil
		})
}

// TestProjectExport verifies that a project can be reimported with the correct settings
func TestProjectExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/19-project/space_creation",
		"../test/terraform/19-project/space_population",
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			// None of these options are used when exporting a space.
			// They are included here to verify they don't affect the exported project.
			ExcludeProjectVariables:       []string{"Test"},
			ExcludeProjectVariablesRegex:  []string{".*"},
			ExcludeProjectVariablesExcept: []string{"DoesNotExist"},
			ExcludeAllProjectVariables:    true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil || len(variables.Variables) != 0 {
						return errors.New("the project must have no variables")
					}

					if strutil.EmptyIfNil(v.Description) != "Test project" {
						return errors.New("The project must be have a description of \"Test project\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if v.AutoCreateRelease {
						return errors.New("The project must not have auto release create enabled")
					}

					if strutil.EmptyIfNil(v.DefaultGuidedFailureMode) != "EnvironmentDefault" {
						return errors.New("The project must be have a DefaultGuidedFailureMode of \"EnvironmentDefault\" (was \"" + strutil.EmptyIfNil(v.DefaultGuidedFailureMode) + "\")")
					}

					if v.DefaultToSkipIfAlreadyInstalled {
						return errors.New("The project must not have DefaultToSkipIfAlreadyInstalled enabled")
					}

					if v.DiscreteChannelRelease {
						return errors.New("The project must not have DiscreteChannelRelease enabled")
					}

					if v.IsDisabled {
						return errors.New("The project must not have IsDisabled enabled")
					}

					if v.IsVersionControlled {
						return errors.New("The project must not have IsVersionControlled enabled")
					}

					if strutil.EmptyIfNil(v.TenantedDeploymentMode) != "Untenanted" {
						return errors.New("The project must be have a TenantedDeploymentMode of \"Untenanted\" (was \"" + strutil.EmptyIfNil(v.TenantedDeploymentMode) + "\")")
					}

					if len(v.IncludedLibraryVariableSetIds) != 0 {
						return errors.New("The project must not have any library variable sets")
					}

					if v.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets {
						return errors.New("The project must not have ProjectConnectivityPolicy.AllowDeploymentsToNoTargets enabled")
					}

					if v.ProjectConnectivityPolicy.ExcludeUnhealthyTargets {
						return errors.New("The project must not have ProjectConnectivityPolicy.AllowDeploymentsToNoTargets enabled")
					}

					if v.ProjectConnectivityPolicy.SkipMachineBehavior != "SkipUnavailableMachines" {
						t.Log("BUG: The project must be have a ProjectConnectivityPolicy.SkipMachineBehavior of \"SkipUnavailableMachines\" (was \"" + v.ProjectConnectivityPolicy.SkipMachineBehavior + "\") - Known issue where the value returned by /api/Spaces-#/ProjectGroups/ProjectGroups-#/projects is different to /api/Spaces-/Projects")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectVarExcludedAllExport verifies that a project can be reimported with excluded variables
func TestProjectVarExcludedAllExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/19-project/space_creation",
		"../test/terraform/19-project/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			ExcludeAllProjectVariables: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil || len(variables.Variables) != 0 {
						return errors.New("The project must not have any variables")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			// check docker feed, used by a step, was exported
			err = func() error {
				collection := octopus.GeneralCollection[octopus.Feed]{}
				err := octopusClient.GetAllResources("Feeds", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.Feed, index int) bool {
					return item.Name == "Docker"
				})) == 0 {
					return errors.New("The feed called \"Docker\" must have been exported")
				}
				return nil
			}()

			return nil
		})
}

// TestProjectVarExcludedExport verifies that a project can be reimported with excluded variables
func TestProjectVarExcludedExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/19-project/space_creation",
		"../test/terraform/19-project/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			ExcludeProjectVariables: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil || len(variables.Variables) != 0 {
						return errors.New("The project must not have any variables")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectVarExcludedRegexExport verifies that a project can be reimported with excluded variables
func TestProjectVarExcludedRegexExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/19-project/space_creation",
		"../test/terraform/19-project/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			ExcludeProjectVariablesRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil || len(variables.Variables) != 0 {
						return errors.New("The project must not have any variables")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectVarExcludedExport verifies that a project can be reimported with excluded variables
func TestProjectVarExcludedExceptExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/19-project/space_creation",
		"../test/terraform/19-project/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			ExcludeProjectVariablesExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil || len(variables.Variables) != 0 {
						return errors.New("The project must not have any variables")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectChannelExport verifies that a project channel can be reimported with the correct settings
func TestProjectChannelExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/20-channel/space_creation",
		"../test/terraform/20-channel/space_population",
		[]string{},
		[]string{"-var=project_test_step_test_package_test_packageid=test2"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					deploymentProcess := octopus.DeploymentProcess{}
					_, err := octopusClient.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(v.DeploymentProcessId), &deploymentProcess)

					if err != nil {
						return err
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].Packages[0].PackageId) != "test2" {
						return errors.New("Deployment process should have renamed the package ID to test2")
					}

					collection := octopus.GeneralCollection[octopus.Channel]{}
					err = octopusClient.GetAllResources("Projects/"+v.Id+"/channels", &collection)

					if err != nil {
						return errors.New(err.Error())
					}

					channelName := "Test"
					foundChannel := false

					for _, c := range collection.Items {
						if c.Name == channelName {
							foundChannel = true

							if strutil.EmptyIfNil(c.Description) != "Test channel" {
								return errors.New("The channel must be have a description of \"Test channel\" (was \"" + strutil.EmptyIfNil(c.Description) + "\")")
							}

							if !c.IsDefault {
								return errors.New("The channel must be be the default")
							}

							if len(c.Rules) != 1 {
								return errors.New("The channel must have one rule")
							}

							if strutil.EmptyIfNil(c.Rules[0].Tag) != "^$" {
								return errors.New("The channel rule must be have a tag of \"^$\" (was \"" + strutil.EmptyIfNil(c.Rules[0].Tag) + "\")")
							}

							if strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].DeploymentAction) != "Test" {
								return errors.New("The channel rule action step must be be set to \"Test\" (was \"" + strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].DeploymentAction) + "\")")
							}

							if strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].PackageReference) != "test" {
								return errors.New("The channel rule action package must be be set to \"test\" (was \"" + strutil.EmptyIfNil(c.Rules[0].ActionPackages[0].PackageReference) + "\")")
							}
						}
					}

					if !foundChannel {
						return errors.New("Project must have an channel called \"" + channelName + "\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTagSetExport verifies that a tag set can be reimported with the correct settings
func TestTagSetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/21-tagset/space_creation",
		"../test/terraform/21-tagset/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.TagSet]{}
			err := octopusClient.GetAllResources("TagSets", &collection)

			if err != nil {
				return err
			}

			resourceName := "tag1"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tagset" {
						return errors.New("The tag set must be have a description of \"Test tagset\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if v.SortOrder != 0 {
						return errors.New("The tag set must be have a sort order of \"0\" (was \"" + fmt.Sprint(v.SortOrder) + "\")")
					}

					tagAFound := false
					for _, u := range v.Tags {
						if u.Name == "a" {
							tagAFound = true

							if strutil.EmptyIfNil(u.Description) != "tag a" {
								return errors.New("The tag a must be have a description of \"tag a\" (was \"" + strutil.EmptyIfNil(u.Description) + "\")")
							}

							if u.Color != "#333333" {
								return errors.New("The tag a must be have a color of \"#333333\" (was \"" + u.Color + "\")")
							}

							if u.SortOrder != 2 {
								return errors.New("The tag a must be have a sort order of \"2\" (was \"" + fmt.Sprint(u.SortOrder) + "\")")
							}
						}
					}

					if !tagAFound {
						return errors.New("Tag Set must have an tag called \"a\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an tag set called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestGitCredentialsExport verifies that a git credential can be reimported with the correct settings
func TestGitCredentialsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/22-gitcredentialtest/space_creation",
		"../test/terraform/22-gitcredentialtest/space_population",
		[]string{},
		[]string{
			"-var=gitcredential_test=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.GitCredentials]{}
			err := octopusClient.GetAllResources("Git-Credentials", &collection)

			if err != nil {
				return err
			}

			resourceName := "test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "test git credential" {
						return errors.New("The git credential must be have a description of \"test git credential\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if v.Details.Username != "admin" {
						return errors.New("The git credential must be have a username of \"admin\" (was \"" + v.Details.Username + "\")")
					}

					if v.Details.Type != "UsernamePassword" {
						return errors.New("The git credential must be have a credential type of \"UsernamePassword\" (was \"" + v.Details.Type + "\")")
					}
				}
			}

			if !found {
				return errors.New("Space must have an git credential called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestScriptModuleExport verifies that a script module set can be reimported with the correct settings
func TestScriptModuleExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/23-scriptmodule/space_creation",
		"../test/terraform/23-scriptmodule/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
			err := octopusClient.GetAllResources("LibraryVariableSets", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test2"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test script module" {
						return errors.New("The library variable set must be have a description of \"Test script module\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					resource := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", v.VariableSetId, &resource)

					if len(resource.Variables) != 2 {
						return errors.New("The library variable set must have two associated variables")
					}

					foundScript := false
					foundLanguage := false
					for _, u := range resource.Variables {
						if u.Name == "Octopus.Script.Module[Test2]" {
							foundScript = true

							if u.Type != "String" {
								return errors.New("The library variable set variable must have a type of \"String\"")
							}

							if strutil.EmptyIfNil(u.Value) != "echo \"hi\"" {
								return errors.New("The library variable set variable must have a value of \"\"echo \\\"hi\\\"\"\"")
							}

							if u.IsSensitive {
								return errors.New("The library variable set variable must not be sensitive")
							}

							if !u.IsEditable {
								return errors.New("The library variable set variable must be editable")
							}
						}

						if u.Name == "Octopus.Script.Module.Language[Test2]" {
							foundLanguage = true

							if u.Type != "String" {
								return errors.New("The library variable set variable must have a type of \"String\"")
							}

							if strutil.EmptyIfNil(u.Value) != "PowerShell" {
								return errors.New("The library variable set variable must have a value of \"PowerShell\"")
							}

							if u.IsSensitive {
								return errors.New("The library variable set variable must not be sensitive")
							}

							if !u.IsEditable {
								return errors.New("The library variable set variable must be editable")
							}
						}
					}

					if !foundLanguage || !foundScript {
						return errors.New("Script module must create two variables for script and language")
					}

				}
			}

			if !found {
				return errors.New("Space must have an library variable set called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTenantsExport verifies that a tenant can be reimported with the correct settings
func TestTenantsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/24-tenants/space_creation",
		"../test/terraform/24-tenants/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTenantsExcept: []string{"Team A"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"Test tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 3 {
						return errors.New("The tenant must have two tags")
					}

					if lo.IndexOf(v.TenantTags, "type/a") == -1 {
						return errors.New("The tenant must have a tag called \"type/a\"")
					}

					if lo.IndexOf(v.TenantTags, "type/b") == -1 {
						return errors.New("The tenant must have a tag called \"type/b\"")
					}

					if lo.IndexOf(v.TenantTags, "type/ignorethis") == -1 {
						return errors.New("The tenant must have a tag called \"type/ignorethis\"")
					}

					if len(v.ProjectEnvironments) != 1 {
						return errors.New("The tenant must be linked to one project")
					}

					for _, u := range v.ProjectEnvironments {
						if len(u) != 3 {
							return errors.New("The tenant must have be linked to three environments")
						}
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTenantsExcludeAllExport verifies that a tenant is excluded
func TestTenantsExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/24-tenants/space_creation",
		"../test/terraform/24-tenants/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTenants: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any tenants in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTenantsExcludeTagsExport verifies that a tenant with excluded tags is not exported, and also that exlcuded
// tags are not exported
func TestTenantsExcludeTagsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/24-tenants/space_creation",
		"../test/terraform/24-tenants/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTenantsWithTags: []string{"type/excluded"},
			ExcludeTenantTags:      []string{"type/ignorethis"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			if lo.SomeBy(collection.Items, func(item octopus.Tenant) bool {
				return item.Name == "Excluded"
			}) {
				return errors.New("Space must have not tenant called \"Excluded\" in space " + recreatedSpaceId)
			}

			teamA := lo.Filter(collection.Items, func(item octopus.Tenant, index int) bool {
				return item.Name == "Team A"
			})

			if len(teamA) != 1 {
				return errors.New("Space must have tenant called \"Team A\" in space " + recreatedSpaceId)
			}

			if lo.SomeBy(teamA[0].TenantTags, func(item string) bool {
				return item == "type/ignorethis"
			}) {
				return errors.New("Tenant must not have a tag called \"type/ignorethis\"")
			}

			tagSetCollection := octopus.GeneralCollection[octopus.TagSet]{}
			err = octopusClient.GetAllResources("TagSets", &tagSetCollection)

			if err != nil {
				return err
			}

			typeTagSet := lo.Filter(tagSetCollection.Items, func(item octopus.TagSet, index int) bool {
				return item.Name == "type"
			})

			if len(typeTagSet) != 1 {
				return errors.New("Space must have a tagset called \"type\"")
			}

			if lo.SomeBy(typeTagSet[0].Tags, func(item octopus.Tag) bool {
				return item.Name == "ignorethis"
			}) {
				return errors.New("Space must not have a tag called \"ignorethis\" in the tag set called \"type\"")
			}

			return nil
		})
}

// TestTenantsExcludeTagSetsExport verifies that excluded tag sets
func TestTenantsExcludeTagSetsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/24-tenants/space_creation",
		"../test/terraform/24-tenants/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTenantTagSets: []string{"type"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			if !lo.SomeBy(collection.Items, func(item octopus.Tenant) bool {
				return item.Name == "Excluded"
			}) {
				return errors.New("Space must have tenant called \"Excluded\" in space " + recreatedSpaceId)
			}

			teamA := lo.Filter(collection.Items, func(item octopus.Tenant, index int) bool {
				return item.Name == "Team A"
			})

			if len(teamA) != 1 {
				return errors.New("Space must have tenant called \"Team A\" in space " + recreatedSpaceId)
			}

			if len(teamA[0].TenantTags) != 0 {
				return errors.New("\"Team A\" must not have any tags")
			}

			tagSetCollection := octopus.GeneralCollection[octopus.TagSet]{}
			err = octopusClient.GetAllResources("TagSets", &tagSetCollection)

			if err != nil {
				return err
			}

			typeTagSet := lo.Filter(tagSetCollection.Items, func(item octopus.TagSet, index int) bool {
				return item.Name == "type"
			})

			if len(typeTagSet) != 0 {
				return errors.New("Space must not have a tagset called \"type\"")
			}

			return nil
		})
}

// TestTenantsExcludeRegexExport verifies that excluded tag sets
func TestTenantsExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/24-tenants/space_creation",
		"../test/terraform/24-tenants/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTenantsRegex: []string{"^Excluded$"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			if lo.SomeBy(collection.Items, func(item octopus.Tenant) bool {
				return item.Name == "Excluded"
			}) {
				return errors.New("Space must not have tenant called \"Excluded\" in space " + recreatedSpaceId)
			}

			teamA := lo.Filter(collection.Items, func(item octopus.Tenant, index int) bool {
				return item.Name == "Team A"
			})

			if len(teamA) != 1 {
				return errors.New("Space must have tenant called \"Team A\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCertificateExport verifies that a certificate can be reimported with the correct settings
func TestCertificateExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/25-certificates/space_creation",
		"../test/terraform/25-certificates/space_population",
		[]string{},
		[]string{
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
			"-var=certificate_tenanted_data=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",
			"-var=certificate_tenanted_password=Password01!",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Certificate]{}
			err := octopusClient.GetAllResources("Certificates", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if v.Notes != "A test certificate" {
						return errors.New("The certificate must be have a description of \"A test certificate\" (was \"" + v.Notes + "\").")
					}

					if v.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The certificate must be have a tenant participation of \"Untenanted\" (was \"" + v.TenantedDeploymentParticipation + "\").")
					}

					if v.SubjectDistinguishedName != "CN=Test Leaf,C=US" {
						return errors.New("The certificate must be have a subject distinguished name of \"CN=Test Leaf,C=US\" (was \"" + v.SubjectDistinguishedName + "\").")
					}

					if len(v.EnvironmentIds) != 0 {
						return errors.New("The certificate must have one project environment.")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The certificate must have no tenant tags.")
					}

					if len(v.TenantIds) != 0 {
						return errors.New("The certificate must have no tenants.")
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			tenantedCertificate := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "Tenanted"
			})

			if len(tenantedCertificate) != 1 {
				return errors.New("Space must have an tenant called \"Tenanted\" in space " + recreatedSpaceId)
			}

			if len(tenantedCertificate[0].TenantIds) != 1 {
				return errors.New("The certificate must have one tenant")
			}

			return nil
		})
}

// TestCertificateExportWithDummyValues verifies that a certificate can be reimported with the correct settings, but
// with dummy values for any secrets.
func TestCertificateExportWithDummyValues(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/66-certificatesdummy/space_creation",
		"../test/terraform/66-certificatesdummy/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			DummySecretVariableValues: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Certificate]{}
			err := octopusClient.GetAllResources("Certificates", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if v.Notes != "A test certificate" {
						return errors.New("The certificate must be have a description of \"A test certificate\" (was \"" + v.Notes + "\")")
					}

					if v.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The certificate must be have a tenant participation of \"Untenanted\" (was \"" + v.TenantedDeploymentParticipation + "\")")
					}

					if v.SubjectDistinguishedName != "CN=test.com" {
						return errors.New("The certificate must be have a subject distinguished name of \"CN=test.com\" (was \"" + v.SubjectDistinguishedName + "\")")
					}

					if len(v.EnvironmentIds) != 0 {
						return errors.New("The certificate must have one environment")
					}

					if len(v.TenantTags) != 0 {
						return errors.New("The tenant must have no tenant tags")
					}

					if len(v.TenantIds) != 0 {
						return errors.New("The tenant must have no tenants")
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTenantVariablesExport verifies that a tenant variables can be reimported with the correct settings
func TestTenantVariablesExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/26-tenant_variables/space_creation",
		"../test/terraform/26-tenant_variables/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := []octopus.TenantVariable{}
			err := octopusClient.GetAllResources("TenantVariables/All", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, tenantVariable := range collection {
				for _, project := range tenantVariable.ProjectVariables {
					if project.ProjectName == resourceName {
						for _, variables := range project.Variables {
							for _, value := range variables {
								// we expect one project variable to be defined
								found = true
								if value != "my value" {
									return errors.New("The tenant project variable must have a value of \"my value\" (was \"" + value + "\")")
								}
							}
						}
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant project variable for the project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestMachinePolicyExport verifies that a machine policies can be reimported with the correct settings
func TestMachinePolicyExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/27-machinepolicy/space_creation",
		"../test/terraform/27-machinepolicy/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.MachinePolicy]{}
			err := octopusClient.GetAllResources("MachinePolicies", &collection)

			if err != nil {
				return err
			}

			resourceName := "Testing"
			found := false
			for _, machinePolicy := range collection.Items {
				if machinePolicy.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(machinePolicy.Description) != "test machine policy" {
						return errors.New("The machine policy must have a description of \"test machine policy\" (was \"" + strutil.EmptyIfNil(machinePolicy.Description) + "\")")
					}

					if machinePolicy.ConnectionConnectTimeout != "00:01:00" {
						return errors.New("The machine policy must have a ConnectionConnectTimeout of \"00:01:00\" (was \"" + machinePolicy.ConnectionConnectTimeout + "\")")
					}

					if *machinePolicy.ConnectionRetryCountLimit != 5 {
						return errors.New("The machine policy must have a ConnectionRetryCountLimit of \"5\" (was \"" + fmt.Sprint(machinePolicy.ConnectionRetryCountLimit) + "\")")
					}

					if machinePolicy.ConnectionRetrySleepInterval != "00:00:01" {
						return errors.New("The machine policy must have a ConnectionRetrySleepInterval of \"00:00:01\" (was \"" + machinePolicy.ConnectionRetrySleepInterval + "\")")
					}

					if machinePolicy.ConnectionRetryTimeLimit != "00:05:00" {
						return errors.New("The machine policy must have a ConnectionRetryTimeLimit of \"00:05:00\" (was \"" + machinePolicy.ConnectionRetryTimeLimit + "\")")
					}

					if machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan != "00:20:00" {
						return errors.New("The machine policy must have a DeleteMachinesElapsedTimeSpan of \"00:20:00\" (was \"" + machinePolicy.MachineCleanupPolicy.DeleteMachinesElapsedTimeSpan + "\")")
					}

					if machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior != "DeleteUnavailableMachines" {
						return errors.New("The machine policy must have a MachineCleanupPolicy.DeleteMachinesBehavior of \"DeleteUnavailableMachines\" (was \"" + machinePolicy.MachineCleanupPolicy.DeleteMachinesBehavior + "\")")
					}

					if machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior != "ExpectedToBeOnline" {
						return errors.New("The machine policy must have a MachineConnectivityPolicy.MachineConnectivityBehavior of \"ExpectedToBeOnline\" (was \"" + machinePolicy.MachineConnectivityPolicy.MachineConnectivityBehavior + "\")")
					}

					if machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType != "Inline" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType of \"Inline\" (was \"" + machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.RunType + "\")")
					}

					if machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody != "" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody of \"\" (was \"" + machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody + "\")")
					}

					if machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType != "Inline" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType of \"Inline\" (was \"" + machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.RunType + "\")")
					}

					if strings.HasPrefix(machinePolicy.MachineHealthCheckPolicy.BashHealthCheckPolicy.ScriptBody, "$freeDiskSpaceThreshold") {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody to start with \"$freeDiskSpaceThreshold\" (was \"" + machinePolicy.MachineHealthCheckPolicy.PowerShellHealthCheckPolicy.ScriptBody + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone) != "UTC" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.HealthCheckCronTimezone of \"UTC\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCronTimezone) + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCron) != "" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.HealthCheckCron of \"\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckCron) + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckType) != "RunScript" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.HealthCheckType of \"RunScript\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckType) + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval) != "00:10:00" {
						return errors.New("The machine policy must have a MachineHealthCheckPolicy.HealthCheckInterval of \"00:10:00\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineHealthCheckPolicy.HealthCheckInterval) + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) != "UpdateOnDeployment" {
						return errors.New("The machine policy must have a MachineUpdatePolicy.CalamariUpdateBehavior of \"UpdateOnDeployment\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) + "\")")
					}

					if strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.TentacleUpdateBehavior) != "NeverUpdate" {
						return errors.New("The machine policy must have a MachineUpdatePolicy.TentacleUpdateBehavior of \"NeverUpdate\" (was \"" + strutil.EmptyIfNil(machinePolicy.MachineUpdatePolicy.CalamariUpdateBehavior) + "\")")
					}
				}
			}

			if !found {
				return errors.New("Space must have an machine policy for the project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectTriggerExport verifies that a project trigger can be reimported with the correct settings
func TestProjectTriggerExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/28-projecttrigger/space_creation",
		"../test/terraform/28-projecttrigger/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundProject := false
			foundTrigger := false
			for _, project := range collection.Items {
				if project.Name == resourceName {
					foundProject = true

					triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
					err = octopusClient.GetAllResources("Projects/"+project.Id+"/Triggers", &triggers)

					for _, trigger := range triggers.Items {
						foundTrigger = true

						if trigger.Name != "test" {
							return errors.New("The project must have a trigger called \"test\" (was \"" + trigger.Name + "\")")
						}

						if trigger.Filter.FilterType != "MachineFilter" {
							return errors.New("The project trigger must have Filter.FilterType set to \"MachineFilter\" (was \"" + trigger.Filter.FilterType + "\")")
						}

						if trigger.Filter.EventGroups[0] != "MachineAvailableForDeployment" {
							return errors.New("The project trigger must have Filter.EventGroups[0] set to \"MachineFilter\" (was \"" + trigger.Filter.EventGroups[0] + "\")")
						}
					}
				}
			}

			if !foundProject {
				return errors.New("Space must have an project \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			if !foundTrigger {
				return errors.New("Project must have a trigger")
			}

			return nil
		})
}

// TestK8sTargetExport verifies that a k8s machine can be reimported with the correct settings
func TestK8sTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/29-k8starget/space_creation",
		"../test/terraform/29-k8starget/space_population",
		[]string{},
		[]string{
			"-var=account_aws_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if strutil.EmptyIfNil(machine.Endpoint.DefaultWorkerPoolId) == "" {
						return errors.New("The machine must specify a default worker pool")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetExcludeExport verifies that a k8s machine can be excluded
func TestK8sTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/29-k8starget/space_creation",
		"../test/terraform/29-k8starget/space_population",
		[]string{},
		[]string{
			"-var=account_aws_account=whatever",
		},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have an targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetExcludeExport verifies that a k8s machine can be excluded
func TestK8sTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/29-k8starget/space_creation",
		"../test/terraform/29-k8starget/space_population",
		[]string{},
		[]string{
			"-var=account_aws_account=whatever",
		},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have an targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetExcludeRegexExport verifies that a k8s machine can be excluded by a regex
func TestK8sTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/29-k8starget/space_creation",
		"../test/terraform/29-k8starget/space_population",
		[]string{},
		[]string{
			"-var=account_aws_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have an targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetExcludeExceptExport verifies that a k8s machine can be excluded by only includes other targets
func TestK8sTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/29-k8starget/space_creation",
		"../test/terraform/29-k8starget/space_population",
		[]string{},
		[]string{
			"-var=account_aws_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {
			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have an targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetAzureAuthExport verifies that a k8s machine with Azure authentication can be reimported with the correct settings
func TestK8sTargetAzureAuthExport(t *testing.T) {

	// need to fix this error:
	// Error: json: cannot unmarshal string into Go struct field KubernetesAzureAuthentication.AdminLogin of type bool
	t.Skip()

	exportSpaceImportAndTest(t,
		"../test/terraform/52-k8stargetazure/space_creation",
		"../test/terraform/52-k8stargetazure/space_population",
		[]string{},
		[]string{"-var=account_azure=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesAzure" {
						return errors.New("Target must use Azure authentication")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.ClusterResourceGroup) != "myresourcegroup" {
						return errors.New("Target must set the resource group to myresourcegroup")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.ClusterName) != "mycluster" {
						return errors.New("Target must set the cluster name to mycluster")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetGcpAuthExport verifies that a k8s machine with google authentication can be reimported with the correct settings
func TestK8sTargetGcpAuthExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/53-k8stargetgcp/space_creation",
		"../test/terraform/53-k8stargetgcp/space_population",
		[]string{},
		[]string{"-var=account_google=secretgoeshere"},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesGoogleCloud" {
						return errors.New("Target must use GCP authentication")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.Project) != "myproject" {
						return errors.New("Target must set the project to myproject")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.ClusterName) != "mycluster" {
						return errors.New("Target must set the cluster name to mycluster")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.Region) != "region" {
						return errors.New("Target must set the region to region")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetTokenAuthExport verifies that a k8s machine with token authentication can be reimported with the correct settings
func TestK8sTargetTokenAuthExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/54-k8stargettoken/space_creation",
		"../test/terraform/54-k8stargettoken/space_population",
		[]string{},
		[]string{
			"-var=account_token=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesStandard" {
						return errors.New("Target must use token authentication")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetCertAuthExport verifies that a k8s machine with certificate authentication can be reimported with the correct settings
func TestK8sTargetCertAuthExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/55-k8stargetcertificate/space_creation",
		"../test/terraform/55-k8stargetcertificate/space_population",
		[]string{},
		[]string{
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesCertificate" {
						return errors.New("Target must use certificate authentication")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSshTargetExport verifies that a ssh machine can be reimported with the correct settings
func TestSshTargetExport(t *testing.T) {
	// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
	exportSpaceImportAndTest(
		t,
		"../test/terraform/30-sshtarget/space_creation",
		"../test/terraform/30-sshtarget/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if machine.Endpoint.Host != "3.25.215.87" {
						return errors.New("The machine must have a Endpoint.Host of \"3.25.215.87\" (was \"" + machine.Endpoint.Host + "\")")
					}

					if machine.Endpoint.DotNetCorePlatform != "linux-x64" {
						return errors.New("The machine must have a Endpoint.DotNetCorePlatform of \"linux-x64\" (was \"" + machine.Endpoint.DotNetCorePlatform + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSshTargetExport verifies that ssh targets can be excluded
func TestSshTargetExcludeAllExport(t *testing.T) {
	// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
	exportSpaceImportAndTest(
		t,
		"../test/terraform/30-sshtarget/space_creation",
		"../test/terraform/30-sshtarget/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSshTargetExcludeExport verifies that ssh targets can be excluded
func TestSshTargetExcludeExport(t *testing.T) {
	// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
	exportSpaceImportAndTest(
		t,
		"../test/terraform/30-sshtarget/space_creation",
		"../test/terraform/30-sshtarget/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSshTargetExcludeRegexExport verifies that ssh targets can be excluded
func TestSshTargetExcludeRegexExport(t *testing.T) {
	// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
	exportSpaceImportAndTest(
		t,
		"../test/terraform/30-sshtarget/space_creation",
		"../test/terraform/30-sshtarget/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSshTargetExcludeExceptExport verifies that ssh targets can be excluded
func TestSshTargetExcludeExceptExport(t *testing.T) {
	// See https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/blob/main/octopusdeploy/schema_ssh_key_account.go#L16
	exportSpaceImportAndTest(
		t,
		"../test/terraform/30-sshtarget/space_creation",
		"../test/terraform/30-sshtarget/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestListeningTargetExport verifies that a listening machine can be reimported with the correct settings
func TestListeningTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/31-listeningtarget/space_creation",
		"../test/terraform/31-listeningtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if machine.Uri != "https://tentacle/" {
						return errors.New("The machine must have a Uri of \"https://tentacle/\" (was \"" + machine.Uri + "\")")
					}

					if machine.Thumbprint != "55E05FD1B0F76E60F6DA103988056CE695685FD1" {
						return errors.New("The machine must have a Thumbprint of \"55E05FD1B0F76E60F6DA103988056CE695685FD1\" (was \"" + machine.Thumbprint + "\")")
					}

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "vm" {
						return errors.New("The machine must have a role of \"vm\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestListeningTargetExcludeAllExport verifies that a listening machine can be excluded
func TestListeningTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/31-listeningtarget/space_creation",
		"../test/terraform/31-listeningtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestListeningTargetExcludeExport verifies that a listening machine can be excluded
func TestListeningTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/31-listeningtarget/space_creation",
		"../test/terraform/31-listeningtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestListeningTargetExcludeExceptExport verifies that a listening machine can be excluded
func TestListeningTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/31-listeningtarget/space_creation",
		"../test/terraform/31-listeningtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestListeningTargetExcludeExceptExport verifies that a listening machine can be excluded
func TestListeningTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/31-listeningtarget/space_creation",
		"../test/terraform/31-listeningtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestPollingTargetExport verifies that a polling machine can be reimported with the correct settings
func TestPollingTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/32-pollingtarget/space_creation",
		"../test/terraform/32-pollingtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if machine.Endpoint.Uri != "poll://abcdefghijklmnopqrst/" {
						return errors.New("The machine must have a Uri of \"poll://abcdefghijklmnopqrst/\" (was \"" + machine.Endpoint.Uri + "\")")
					}

					if machine.Thumbprint != "1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86" {
						return errors.New("The machine must have a Thumbprint of \"1854A302E5D9EAC1CAA3DA1F5249F82C28BB2B86\" (was \"" + machine.Thumbprint + "\")")
					}

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "vm" {
						return errors.New("The machine must have a role of \"vm\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestPollingTargetExcludeAllExport verifies that a polling machine can be excluded
func TestPollingTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/32-pollingtarget/space_creation",
		"../test/terraform/32-pollingtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestPollingTargetExcludeExport verifies that a polling machine can be excluded
func TestPollingTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/32-pollingtarget/space_creation",
		"../test/terraform/32-pollingtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestPollingTargetExcludeExceptExport verifies that a polling machine can be excluded
func TestPollingTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/32-pollingtarget/space_creation",
		"../test/terraform/32-pollingtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestPollingTargetExcludeRegexExport verifies that a polling machine can be excluded
func TestPollingTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/32-pollingtarget/space_creation",
		"../test/terraform/32-pollingtarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.PollingEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCloudRegionTargetExport verifies that a cloud region can be reimported with the correct settings
func TestCloudRegionTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/33-cloudregiontarget/space_creation",
		"../test/terraform/33-cloudregiontarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "cloud" {
						return errors.New("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCloudRegionTargetExcludeAllExport verifies that a cloud region can be excluded
func TestCloudRegionTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/33-cloudregiontarget/space_creation",
		"../test/terraform/33-cloudregiontarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCloudRegionTargetExcludeExport verifies that a cloud region can be excluded
func TestCloudRegionTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/33-cloudregiontarget/space_creation",
		"../test/terraform/33-cloudregiontarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCloudRegionTargetExcludeRegexExport verifies that a cloud region can be excluded
func TestCloudRegionTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/33-cloudregiontarget/space_creation",
		"../test/terraform/33-cloudregiontarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestCloudRegionTargetExcludeExceptExport verifies that a cloud region can be excluded
func TestCloudRegionTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/33-cloudregiontarget/space_creation",
		"../test/terraform/33-cloudregiontarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.CloudRegionResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestOfflineDropTargetExport verifies that an offline drop can be reimported with the correct settings
func TestOfflineDropTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/34-offlinedroptarget/space_creation",
		"../test/terraform/34-offlinedroptarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "offline" {
						return errors.New("The machine must have a role of \"offline\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}

					if machine.Endpoint.ApplicationsDirectory != "c:\\temp" {
						return errors.New("The machine must have a Endpoint.ApplicationsDirectory of \"c:\\temp\" (was \"" + machine.Endpoint.ApplicationsDirectory + "\")")
					}

					if machine.Endpoint.OctopusWorkingDirectory != "c:\\temp" {
						return errors.New("The machine must have a Endpoint.OctopusWorkingDirectory of \"c:\\temp\" (was \"" + machine.Endpoint.OctopusWorkingDirectory + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestOfflineDropTargetExcludeExport verifies that an offline drop can be excluded
func TestOfflineDropTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/34-offlinedroptarget/space_creation",
		"../test/terraform/34-offlinedroptarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargets: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestOfflineDropTargetExcludeExceptExport verifies that an offline drop can be excluded
func TestOfflineDropTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/34-offlinedroptarget/space_creation",
		"../test/terraform/34-offlinedroptarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestOfflineDropTargetExcludeRegexExport verifies that an offline drop can be excluded
func TestOfflineDropTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/34-offlinedroptarget/space_creation",
		"../test/terraform/34-offlinedroptarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestOfflineDropTargetExcludeAllExport verifies that an offline drop can be excluded
func TestOfflineDropTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/34-offlinedroptarget/space_creation",
		"../test/terraform/34-offlinedroptarget/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureCloudServiceTargetExport verifies that a azure cloud service target can be reimported with the correct settings
func TestAzureCloudServiceTargetExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/35-azurecloudservicetarget/space_creation",
		"../test/terraform/35-azurecloudservicetarget/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=MIIQFgIBAzCCD9wGCSqGSIb3DQEHAaCCD80Egg/JMIIPxTCCBdcGCSqGSIb3DQEHBqCCBcgwggXEAgEAMIIFvQYJKoZIhvcNAQcBMBwGCiqGSIb3DQEMAQYwDgQID45832+aYFECAggAgIIFkIyL07gJLw9QA/WpRBhh+eDpKQ7/R4ZX7uOKch6UCl+JYs9BE2TkVHSrukZ8YeQY5aHRK6kB5fZSzyD+7+cNIEx1RU8owOdOl0uSIUvDuUu6MXgcCNwkZ+I9DV6eqU991RiO2B2kGAHKsr28z0voOXGP9JDG2kJMF+R1j8LQqRCkJUrEaX0BJ1zeuY05Cv3domuevKSq+Xg1VRGWZEc01Iprb2gTpMpFwusLGase4pyka9XCbDIkqUEjt65cOXzjyiZnKgX376wA6TfB+xrmC9g/839Rt6V2pZA84bDB0AcwHMyUjXN9mdB1mIfFRvOID8Pp019Oc7B+cfipZTubIZld6BPDFRiE8yd3ixkQSPTDv5eHYxtUEt969M6h1viE5xF1YvJ3RaxkZXIOTx5kel5XtOQbMaF8W+pzoY7ljl9bjN+0nPSJgApcTxCvYoILv9Ecy/Ry8CH91BTNTJr+rdLNtcMGFskrS2U+wUhuMtMeEkAPVX2BWYjWvnlDsXiwpzoV/fpzmZCqD8q03Tzt/gM/IaxX3Eb/MZdB60FepgxHu7oom5IQMCzgymUsq4jtKD4fQdBu+QVVggoB1hlrDomCfraThieBqTpQBSTW3TpQ2gPq2pFAIAqexXd7kQVouWDuQWa8vXU35SHKbE3l8yrVT3pK7EdBT4+YQfYYXUpGnnbUuFq26oTV1B1NmVg9bOMYOnEIMBo4ZfPhaMU+VqFiEHVTQ/khhqsPAvscaIArBwAuQGNNuaV0GWHR7qztGeJFMRoqmyKb+Pxzcue6Z5QVaCMg9t1kFaTMdiomA7W6VYww8euCx1kiMjiczC2DTxamp1B4+bQBJQsSGJjhbe1EOMYRRauYhWPUpbF5kGkp7HwRdT6W9dDvs987dLR90jwOuBfmshdVabVuQI8kxglS8SSYG4oSbhIOmz88ssjeQlNCU92DpHHW52+Rvyxp5vitFwpfs1niZRBSCTwMvA2kqaU7MlgDq+jjgPHLP0YL7K72zbYE5aVTT5C7tc8jwwJ1XiRNyO8aRClSN099rTfRxUrxekIP+hOYVfiMIBvtuG+BotIEGlykKjC21W0f4zFKMjmiz7MKnhSpcUO2FgjKZlXi8haGYNRKBmPXNF7Xs+dsT6zv1IUN8/ssrLITpVk6DRAAhBGHt64XHRQql4EqeCO4fPemUBQ1IQOFy17krSWfvqRgEi+lTBVh3JWRNBbQq2ZSF2LFFy0sdsEyAzRDgeg5p8zCTu1HuXV7WMZwkme2RnqaU9/6qF9SlGPtgagwDRxAjsljA531RG0s+Mo3z8tAoHLn66s7Di/VNho5WnlcfR0FAMCfG/JROjOCLPDyxNsuIHRah/V3g/jsNkmomXutDwBiiiV6Cfl6fMwf+xPNA5JvrYTyaGVdxxrLz0YyYbdmzbaFFSSN4Xtmi6TrotGzRdeHj6uFT24H7xonJtSzNi7+mWuU2/r4SNATVIJ9yHxAiGgrfVTMFi98zV9eor5mtWMf6exGE9Fs0iIdPDYb0le6/69jeH1mpGQ3HTyLQlaEo4OPeDsLYm7jyrk6jxTN/NEZEXO7ify/7AJIRK7Dv5hR5h2C2u70/VWtIB5kozDz53lmOMzSeKLvG0lvCm1jcvB12SVlnJjAnmy8vFLiLyLxTRftC0nlv14LB1pl+h5EIWWn0/kGCUk57rOYmzwVo59nck8pyQN/q6Nwnijw27tT2FG79Qjhxzeproe3U6i48elCU/mdUSBhqP4jTiacV+lU8tFGVESZpV/Pkxan+aNT73QeiqbMFW4eiyqpqPiYx1QiNRAoGy7qJOriaDgLkOnLtwpA+dVTs663abR1h868j+pt6g4CjiYBGcugALF0lrCR65fvBaRbs8PpthSmNUO7iAJLKkz+m56lMIIJ5gYJKoZIhvcNAQcBoIIJ1wSCCdMwggnPMIIJywYLKoZIhvcNAQwKAQKgggluMIIJajAcBgoqhkiG9w0BDAEDMA4ECEkD2RX/XiDvAgIIAASCCUjX1gBBLbSZDI0UnWXYGo+ro+gdXHzpueUbw4McGRMfofZqkb5+ksbDat1UY6PbtIunwvxP08mlRCfJWOqNg1XGNP7FYCuzmXv+sOeEwvRMsEf+0ufI0cGHTYneAFef94stFB5Q73zGtO60KvwcjVKRBJwnDbWM61x6KN9681sC3WTaS163VtUNmuntw+WU3DNKcvXuCUhOLjciqcwoa939TpL1UkK7VTIKMZTHmlylKPy5MQvSYM0JjHl/yuZeQ5IldwMH8Ts0JwBvaC47za5S2P+7c2dzl8kI0Wafqxd7a+uwf9DWmgVC0J6oaR+kmMeuTJG+ggiQ87i1+m16m+5nhVdlwtVKYABSlSPnlDgoel33QWzfy7RSug+YDk/8JEKS0slrNe94e20gyIeEzxaNaM+rjJ2MDgkNhb7NxGZdR1oOreAafpPZ1UansKhHqvUeWbQ/rUGdk/8TbehiiX2Jlh7F3NbbsYT/6zMvK/Zq8gS0FrGZxA1bFDApd+5m4qinzbedctac++8keijuISTq+t257hr3I4+4jDHhwoGN3qE1zlAQj5NDc4qb3QM47frEY6ENwyNWjrzeGGI3tphYwpIq2ocufqJjgYR9TcQPQEURA+35opmoHzy+68iPJoZT0bqFx/OSwQP0JC1OMNAtMjZTswVF/GX6GeRk6iF2FNTMIQ/DunvMTooVxupjaujFCxfnM2p8fuz/De4ciTVqg1B4bdk+upPzgAYFgKl9ynGbeHLQQq0ETSfmxxc7YIwrJ1UsWECIENe1ZZG4texjYE14nql7crx8rT4lqzcRAuyfJ8y/nCwXtPGGqT34AJfmGZEFKrX+i8c5jUTreSXdI4FoDIW8L2/o5zJv/wqQd0s0ly0DUCbqZ8DE2WXpN8iReM5u1GJP7xHbeJg3lkqSo2R4HTv1bV/E25aTdacwRsd5IkBZnAJejZKhwmVhga2cfnHuqxL1o6h+l6qygTtVdis1Pu7xg5RoeaVRsdzBpHKQ3mL/jfMnAccOIoCe45mMmN6ZOVWqVFNAyYbjwYoOG/zgyawsx/FTQB166+xZw0vH3Jj1Wd80wpQX47QMvRb1LOfe680p/mt5JrUkN8yuepOKCseJUEmZO+OxaNe0N1MnLdGLKtYncu25FOosMDRvw+DKQtDtfEGyKPJNWdrU7C9swQ29GarclrDwbqo0Ris93SWfx5tCJD6vHCAnV3u6A2eWFZfKqMDC6hkLlnMLoStehbfTzvSuyvK7vbq+VMmACx6EpP8PDxf5G5/RJFGHAOZWT1tEl2mPIQSvgMO/o23S8HKzCRelYuSdz3iqufYZphVuNKFyMNIc363lImgAqOMMo1JrFu3UBUlqjUllhqlKq6ZDcG6jfNipo1XEgt1gs824JsECHg8xsVKJ+bhY1yK92kh4u2rSRtahOFiU0z4CipkmtP9KvrQqnQX65+FLEJ7/DSKF82c5dUIBWw/NJlgsHTs4utL3+An2EwMYgRGtESOX0PQWH20GczzbFOxDYfdi0/AVtoKkwjo60PCIznOnPzTi527zNggfnXv6t15WDVPIC8yjn/4GJIEaeWpTNZL8Ff3R1BMD08QZEY1Ucal1adWUxKtBnmxvt/FlkkSPnbgGxWm0eWeU10+zNLnPL0Zr7jWNtmJFhONvmr4xbqZsvWzDJeHmKYMRs4l67Yt+/Pgh6p2U0uDlT7pCYi6KTsrOLeZOEB0BRwHXt1ks9cs1JDS4nfDA/9a6NOGErKRtvy0rMwshN3e/jj3g6GdRh2RSRNHIffCsf3QN3k3saLvnniK992898CrH4W47SysFUbiP+ukdX8pvarpN+aeKtxc7uvzcBJKBdW1jvpsJBDMRd6OrGnuei+LSNcCyVdrUQc7c1Gcnl8jkEl2wUcyDkZP4ZZuK2PFRPVIQJ0dgRdFvgjzridSzO4PPTNuTbX68Y4aNtE/pAKzJlAlE/xNHtJLXOwWUxmfC4crNEW0ihAByUaGGu8Owgm2mzAwKHQmAie90GN1ov9oHU+6tBPNIL6Xbcf6roqf3VFh6Z8lz1vAWci/qG7Pf+LCf+HTDqI6nba/ihbO2AxxAy8WdY+tNkJtc5giRjdE2Rha9y5aFznQM/AyiB+iiuA8i6tQWvu1v8uxdzbaCfckq/NvYp+E8wCTRsZ1KnK19g5e8ltAOmsokW4MU1femcOJNwJWXB7O41ihJNo3rYCxe+sQ2lmfVcOyONnuB/woxBEuoOQJGV50VCWBFCCTFQe9tlXEHqVqJSXoXZbpPDluMFGoVxogUCtcFpJy/uS8R/XqitBdcsIWcXMXOkQGLCdA+FYeiwdARUptIInl1pSaZNkn7CwOaIWc0/cXjTYrxFodkQ/sPU2AB8IW641AQjg2yiCDFx7penax8hvUEVK/jxwQZYqwCrkYme4t77EMwquCo0JUXkypy19fpuXm/qiin9oErJ2LZg5SEqhReskipKJIZRb1tRCFFZm5QWM0e6cOHI2nl16irUGbuojGpYVbdJdCW4XLz82rv1kCdYZGXXtKs8F8Uayz6b7Jklksx39bVtwQq7gF/1KfEyO3f3kn6ASOjG/IigBcvygERycx9rDOf0coGLNEaxSsX8UE3EDHKqqZCWYxXOPBotTMWxucYYq94dm6qrn/LuflFyDkuryU000r5Cw9ZnnnfwuK4TBSV+8wgJhLvwrqnixLbsg3r3iydFdRJcCqlbP4iCO9uV/Wx3ybUD6OttVfamsKXGE4546d/tbRPstI2yb2U+XfuC/7jMaDEn9mhZYZKMm4mU1SLy/Xd/QfzKrshd/fwo+4ytqzn5pUQsz1dwPnWqcAZ/rqdC2Sduu0DzV4JxCSdIqV3ly+ddmbHN8CqraVK6wVU6c/MAQWIJtHJGzyaFuTP5o6+NKU3bL7mn81K6ERRa26rrGJ1m4wcaZ2DQz7tPjGXvgyf4C/G0kHe044uugF5o/JbeTWIBS5MEN7LwzAHq+hRZtn3gS7CVa9RuKv83CXAxXKGyVWhhH1I1/1hg4D9g/SId5oKFoX/4uwHU3qL2TR5x2IudbAsM5aR07WkdH6AlYR39uHYJD0YbSetGpPwB8kE9UxUf3OapTPZ0H3BsK8e3gmPeeV5HZdNhLQyooSeZrBCBMrHZPWKBd5lyJ+A55eXlZ3Ipjvga7oxSjAjBgkqhkiG9w0BCRQxFh4UIBwAdABlAHMAdAAuAGMAbwBtIB0wIwYJKoZIhvcNAQkVMRYEFOMGhtI87uqZJdmGtKQ0ocH8zuq9MDEwITAJBgUrDgMCGgUABBT/J2cWVPSNRgxssWAizswpxhPtlgQI/Z6OnKgtwf4CAggA",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Azure"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Azure"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "cloud" {
						return errors.New("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}

					if machine.Endpoint.CloudServiceName != "servicename" {
						return errors.New("The machine must have a Endpoint.CloudServiceName of \"c:\\temp\" (was \"" + machine.Endpoint.CloudServiceName + "\")")
					}

					if machine.Endpoint.StorageAccountName != "accountname" {
						return errors.New("The machine must have a Endpoint.StorageAccountName of \"accountname\" (was \"" + machine.Endpoint.StorageAccountName + "\")")
					}

					if !machine.Endpoint.UseCurrentInstanceCount {
						return errors.New("The machine must have Endpoint.UseCurrentInstanceCount set")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureCloudServiceTargetExcludeExport verifies that a azure cloud service target can be excluded
func TestAzureCloudServiceTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/35-azurecloudservicetarget/space_creation",
		"../test/terraform/35-azurecloudservicetarget/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=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",
		},
		args2.Arguments{
			ExcludeTargets: []string{"Azure"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureCloudServiceTargetExcludeRegexExport verifies that a azure cloud service target can be excluded
func TestAzureCloudServiceTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/35-azurecloudservicetarget/space_creation",
		"../test/terraform/35-azurecloudservicetarget/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=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",
		},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureCloudServiceTargetExcludeExceptExport verifies that a azure cloud service target can be excluded
func TestAzureCloudServiceTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/35-azurecloudservicetarget/space_creation",
		"../test/terraform/35-azurecloudservicetarget/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=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",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureCloudServiceTargetExcludeAllExport verifies that a azure cloud service target can be excluded
func TestAzureCloudServiceTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/35-azurecloudservicetarget/space_creation",
		"../test/terraform/35-azurecloudservicetarget/space_population",
		[]string{},
		[]string{
			"-var=account_subscription_cert=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",
		},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureServiceFabricTargetExport verifies that a service fabric target can be reimported with the correct settings
func TestAzureServiceFabricTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/36-servicefabrictarget/space_creation",
		"../test/terraform/36-servicefabrictarget/space_population", []string{
			"-var=target_service_fabric=whatever",
		}, []string{
			"-var=target_service_fabric=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Service Fabric"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Service Fabric"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "cloud" {
						return errors.New("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}

					if machine.Endpoint.ConnectionEndpoint != "http://endpoint" {
						return errors.New("The machine must have a Endpoint.ConnectionEndpoint of \"http://endpoint\" (was \"" + machine.Endpoint.ConnectionEndpoint + "\")")
					}

					if machine.Endpoint.AadCredentialType != "UserCredential" {
						return errors.New("The machine must have a Endpoint.AadCredentialType of \"UserCredential\" (was \"" + machine.Endpoint.AadCredentialType + "\")")
					}

					if machine.Endpoint.AadUserCredentialUsername != "username" {
						return errors.New("The machine must have a Endpoint.AadUserCredentialUsername of \"username\" (was \"" + machine.Endpoint.AadUserCredentialUsername + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureServiceFabricTargetExcludeExport verifies that a service fabric target can be excluded
func TestAzureServiceFabricTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/36-servicefabrictarget/space_creation",
		"../test/terraform/36-servicefabrictarget/space_population", []string{
			"-var=target_service_fabric=whatever",
		}, []string{},
		args2.Arguments{
			ExcludeTargets: []string{"Service Fabric"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no target in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureServiceFabricTargetExcludeRegexExport verifies that a service fabric target can be excluded
func TestAzureServiceFabricTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/36-servicefabrictarget/space_creation",
		"../test/terraform/36-servicefabrictarget/space_population", []string{
			"-var=target_service_fabric=whatever",
		}, []string{},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no target in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureServiceFabricTargetExcludeExceptExport verifies that a service fabric target can be excluded
func TestAzureServiceFabricTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/36-servicefabrictarget/space_creation",
		"../test/terraform/36-servicefabrictarget/space_population", []string{
			"-var=target_service_fabric=whatever",
		}, []string{},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no target in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureServiceFabricTargetExcludeAllExport verifies that a service fabric target can be excluded
func TestAzureServiceFabricTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/36-servicefabrictarget/space_creation",
		"../test/terraform/36-servicefabrictarget/space_population", []string{
			"-var=target_service_fabric=whatever",
		}, []string{},
		args2.Arguments{
			ExcludeAllTargets: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no target in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureWebAppTargetExport verifies that a web app target can be reimported with the correct settings
func TestAzureWebAppTargetExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/37-webapptarget/space_creation",
		"../test/terraform/37-webapptarget/space_population",
		[]string{
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"Web App"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Web App"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if len(machine.Roles) != 1 {
						return errors.New("The machine must have 1 role")
					}

					if machine.Roles[0] != "cloud" {
						return errors.New("The machine must have a role of \"cloud\" (was \"" + machine.Roles[0] + "\")")
					}

					if machine.TenantedDeploymentParticipation != "Untenanted" {
						return errors.New("The machine must have a TenantedDeploymentParticipation of \"Untenanted\" (was \"" + machine.TenantedDeploymentParticipation + "\")")
					}

					if machine.Endpoint.ResourceGroupName != "mattc-webapp" {
						return errors.New("The machine must have a Endpoint.ResourceGroupName of \"mattc-webapp\" (was \"" + machine.Endpoint.ResourceGroupName + "\")")
					}

					if machine.Endpoint.WebAppName != "mattc-webapp" {
						return errors.New("The machine must have a Endpoint.WebAppName of \"mattc-webapp\" (was \"" + machine.Endpoint.WebAppName + "\")")
					}

					if machine.Endpoint.WebAppSlotName != "slot1" {
						return errors.New("The machine must have a Endpoint.WebAppSlotName of \"slot1\" (was \"" + machine.Endpoint.WebAppSlotName + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureWebAppTargetExcludeExport verifies that a web app target can be excluded
func TestAzureWebAppTargetExcludeExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/37-webapptarget/space_creation",
		"../test/terraform/37-webapptarget/space_population",
		[]string{
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{
			ExcludeTargets: []string{"Web App"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureWebAppTargetExcludeRegexExport verifies that a web app target can be excluded
func TestAzureWebAppTargetExcludeRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/37-webapptarget/space_creation",
		"../test/terraform/37-webapptarget/space_population",
		[]string{
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsRegex: []string{".*"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureWebAppTargetExcludeRegexExport verifies that a web app target can be excluded
func TestAzureWebAppTargetExcludeExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/37-webapptarget/space_creation",
		"../test/terraform/37-webapptarget/space_population",
		[]string{
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{
			ExcludeTargetsExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestAzureWebAppTargetExcludeAllExport verifies that a web app target can be excluded
func TestAzureWebAppTargetExcludeAllExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/37-webapptarget/space_creation",
		"../test/terraform/37-webapptarget/space_population",
		[]string{
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{
			ExcludeAllTargets:        true,
			IncludeOctopusOutputVars: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			testFramework := test.OctopusContainerTest{}
			serverUrl, err := testFramework.GetOutputVariable(t, terraformStateDir, "octopus_server")

			if err != nil {
				return err
			}

			if serverUrl == "" {
				return errors.New("The project must have created an output variable called octopus_server")
			}

			octopusSpaceName, err := testFramework.GetOutputVariable(t, terraformStateDir, "octopus_space_name")

			if err != nil {
				return err
			}

			if octopusSpaceName == "" {
				return errors.New("The project must have created an output variable called octopusSpaceName")
			}

			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
			err = octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must have no targets in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSingleProjectGroupExport verifies that a single project can be reimported with the correct settings.
// This is one of the larger tests, verifying that the graph of resources linked to a project have been exported,
// and that unrelated resources were not exported.
func TestSingleProjectGroupExport(t *testing.T) {
	if os.Getenv("GIT_CREDENTIAL") == "" {
		t.Fatalf("the GIT_CREDENTIAL environment variable must be set to a GitHub access key")
	}

	exportProjectImportAndTest(t,
		"Test",
		"../test/terraform/38-multipleprojects/space_creation",
		"../test/terraform/38-multipleprojects/space_population",
		"../test/terraform/z-createspace",
		[]string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
		},
		[]string{},
		[]string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
			"-var=project_test_git_base_path=.octopus/integrationtestimport",
			"-var=feed_helm_password=whatever",
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
			"-var=account_gke=Password01!",
		},
		args2.Arguments{
			ExcludeVariableEnvironmentScopes: []string{"Test"},
			ExcludeProjectVariables:          []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Test that the project exported its project group
			err := func() error {
				collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
				err := octopusClient.GetAllResources("ProjectGroups", &collection)

				if err != nil {
					return err
				}

				found := false
				for _, v := range collection.Items {
					if v.Name == "Test" {
						found = true
						if strutil.EmptyIfNil(v.Description) != "Test Description" {
							return errors.New("The project group must be have a description of \"Test Description\"")
						}
					}
				}

				if !found {
					return errors.New("Space must have a project group called \"Test\"")
				}
				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single project was exported
			err = func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err = octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				excluded := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "Test"
				})

				if len(excluded) != 0 {
					return errors.New("The variable called Test shoudl be excluded)")
				}

				envScoped := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "Test2"
				})

				tenantScoped := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "tenantscoped"
				})

				environmentScopedButRemoved := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "Test3"
				})

				helmFeed := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "HelmFeed"
				})

				usernamePassword := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "UsernamePassword"
				})

				workerPool := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "WorkerPool"
				})

				certificate := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "Certificate"
				})

				emptyVar := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "TestNull"
				})

				if len(emptyVar) == 0 {
					return errors.New("The project must have 1 variable called \"TestNull\"")
				}

				if len(envScoped) == 0 {
					return errors.New("The project must have 1 variable called \"Test2\"")
				}

				if len(tenantScoped) == 0 {
					return errors.New("The project must have 1 variable called \"tenantscoped\"")
				}

				if len(envScoped[0].Scope.Environment) != 1 {
					return errors.New("The project must have 1 variable called \"Test2\" scoped to an environment")
				}

				if len(environmentScopedButRemoved[0].Scope.Environment) != 0 {
					return errors.New("The project must have 1 variable called \"Test3\" with the environment scope removed due to the ExcludeVariableEnvironmentScopes setting")
				}

				if len(tenantScoped[0].Scope.TenantTag) != 1 {
					return errors.New("The project must have 1 variable called \"tenantscoped\" scoped to an tagset")
				}

				if len(helmFeed) != 1 {
					return errors.New("The project must have 1 variable called \"HelmFeed\"")
				}

				if err != nil {
					return err
				}

				// Check the helm feed, used by a variable, was exported
				err = func() error {
					helmFeedResource := octopus.Feed{}
					_, err = octopusClient.GetSpaceResourceById("Feeds", strutil.EmptyIfNil(helmFeed[0].Value), &helmFeedResource)

					if err != nil {
						return err
					}

					if helmFeedResource.Name != "Helm" {
						return errors.New("The feed called \"Helm\" must have been exported")
					}

					return nil
				}()

				if err != nil {
					return err
				}

				if len(usernamePassword) != 1 {
					return errors.New("The project must have 1 variable called \"UsernamePassword\"")
				}

				// check rge account, used by a variable, was exported
				err = func() error {
					accountResource := octopus.Account{}
					_, err = octopusClient.GetSpaceResourceById("Accounts", strutil.EmptyIfNil(usernamePassword[0].Value), &accountResource)

					if err != nil {
						return err
					}

					if accountResource.Name != "GKE" {
						return errors.New("The account called \"GKE\" must have been exported")
					}

					return nil
				}()

				if err != nil {
					return err
				}

				if len(workerPool) != 1 {
					return errors.New("The project must have 1 variable called \"WorkerPool\"")
				}

				// check rge worker pool, used by a variable, was exported
				err = func() error {
					workerPoolResource := octopus.WorkerPool{}
					_, err = octopusClient.GetSpaceResourceById("WorkerPools", strutil.EmptyIfNil(workerPool[0].Value), &workerPoolResource)

					if err != nil {
						return err
					}

					if workerPoolResource.Name != "Docker" {
						return errors.New("The worker pool called \"Docker\" must have been exported")
					}

					return nil
				}()

				if err != nil {
					return err
				}

				if len(certificate) != 1 {
					return errors.New("The project must have 1 variable called \"Certifcate\"")
				}

				// check the certificate, used by a variable, was exported
				err = func() error {
					certificateResource := octopus.Certificate{}
					_, err = octopusClient.GetSpaceResourceById("Certificates", strutil.EmptyIfNil(certificate[0].Value), &certificateResource)

					if err != nil {
						return err
					}

					if certificateResource.Name != "Test" {
						return errors.New("The certificate called \"Test\" must have been exported")
					}

					return nil
				}()

				if err != nil {
					return err
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single channel was exported
			err = func() error {
				channelsCollection := octopus.GeneralCollection[octopus.Channel]{}
				err = octopusClient.GetAllResources("Channels", &channelsCollection)

				if err != nil {
					return err
				}

				foundChannel := false
				for _, v := range channelsCollection.Items {
					if v.Name == "Test 1" {
						foundChannel = true
					}

					if v.Name == "Test 2" {
						return errors.New("The second channel must not have been exported")
					}
				}

				if !foundChannel {
					return errors.New("The space must have a channel called \"Test 1\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single trigger was exported
			err = func() error {
				triggersCollection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
				err = octopusClient.GetAllResources("ProjectTriggers", &triggersCollection)

				if err != nil {
					return err
				}

				foundTrigger := false
				for _, v := range triggersCollection.Items {
					if v.Name == "Test 1" {
						foundTrigger = true
					}

					if v.Name == "Test 2" {
						return errors.New("The second trigger must not have been exported")
					}
				}

				if !foundTrigger {
					return errors.New("The space must have a trigger called \"Test 1\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single tenant was exported
			err = func() error {
				tenantsCollection := octopus.GeneralCollection[octopus.Tenant]{}
				err = octopusClient.GetAllResources("Tenants", &tenantsCollection)

				if err != nil {
					return err
				}

				foundTenant := false
				for _, v := range tenantsCollection.Items {
					if v.Name == "Team A" {
						foundTenant = true
					}

					if v.Name == "Team B" {
						return errors.New("The second tenant must not have been exported")
					}
				}

				if !foundTenant {
					return errors.New("The space must have a tenant called \"Team A\"")
				}
				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the tenant tags were exported
			err = func() error {
				tagsCollection := octopus.GeneralCollection[octopus.TagSet]{}
				err = octopusClient.GetAllResources("TagSets", &tagsCollection)

				if err != nil {
					return err
				}

				foundTag := false
				for _, v := range tagsCollection.Items {
					if v.Name == "tag1" {
						foundTag = true
					}

					if v.Name == "tag2" {
						return errors.New("The space must not have a tagset called \"tag2\"")
					}
				}

				if !foundTag {
					return errors.New("The space must have a tagset called \"tag1\"")
				}
				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the environments were exported
			err = func() error {
				environmentsCollection := octopus.GeneralCollection[octopus.Tenant]{}
				err = octopusClient.GetAllResources("Environments", &environmentsCollection)

				if err != nil {
					return err
				}

				foundEnvironmentDev := false
				foundEnvironmentTest := false
				foundEnvironmentProduction := false
				for _, v := range environmentsCollection.Items {
					if v.Name == "Development" {
						foundEnvironmentDev = true
					}

					if v.Name == "Test" {
						foundEnvironmentTest = true
					}

					if v.Name == "Production" {
						foundEnvironmentProduction = true
					}

					if v.Name == "Blah" {
						return errors.New("The environment called \"Blah\" must not been exported")
					}
				}

				if !foundEnvironmentDev {
					return errors.New("The space must have a space called \"Deveopment\"")
				}

				if !foundEnvironmentTest {
					return errors.New("The space must have a space called \"Test\"")
				}

				if !foundEnvironmentProduction {
					return errors.New("The space must have a space called \"Production\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the library variable set was exported
			err = func() error {
				libraryVariableSetCollection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
				err = octopusClient.GetAllResources("LibraryVariableSets", &libraryVariableSetCollection)

				if err != nil {
					return err
				}

				foundLibraryVariableSet := false
				for _, v := range libraryVariableSetCollection.Items {
					if v.Name == "Test" {
						foundLibraryVariableSet = true
					}

					if v.Name == "Test2" {
						return errors.New("The library variable set called \"Test2\" must not been exported")
					}
				}

				if !foundLibraryVariableSet {
					return errors.New("The space must have a library variable called \"Test\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the library variable set was exported
			err = func() error {
				collection := octopus.GeneralCollection[octopus.Lifecycle]{}
				err = octopusClient.GetAllResources("Lifecycles", &collection)

				if err != nil {
					return err
				}

				found := false
				for _, v := range collection.Items {
					if v.Name == "Simple" {
						found = true
					}

					if v.Name == "Simple2" {
						return errors.New("The lifecycle called \"Simple2\" must not been exported")
					}
				}

				if !found {
					return errors.New("The space must have a lifecycle called \"Simple\"")
				}

				return nil
			}()

			// Verify that the git credential was exported
			err = func() error {
				collection := octopus.GeneralCollection[octopus.GitCredentials]{}
				err = octopusClient.GetAllResources("Git-Credentials", &collection)

				if err != nil {
					return err
				}

				found := false
				for _, v := range collection.Items {
					if v.Name == "matt" {
						found = true
					}
				}

				if !found {
					return errors.New("The space must have a git credential called \"matt\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectLookupExport verifies that a single project can be reimported with the correct settings.
// This is one of the larger tests, verifying that the graph of resources linked to a project have been referenced via data source lookups,
// and that unrelated or excluded resources were not exported.
func TestSingleProjectLookupExport(t *testing.T) {
	if os.Getenv("GIT_CREDENTIAL") == "" {
		t.Fatalf("the GIT_CREDENTIAL environment variable must be set to a GitHub access key")
	}

	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/43-multipleprojectslookup/space_creation",
		"../test/terraform/43-multipleprojectslookup/space_prepopulation",
		"../test/terraform/43-multipleprojectslookup/space_population",
		"../test/terraform/43-multipleprojectslookup/space_creation",
		"../test/terraform/43-multipleprojectslookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
		},
		[]string{
			"-var=project_test_git_base_path=.octopus/integrationtestimport",
		},
		args2.Arguments{
			ExcludeTenants:                  []string{"Team A"},
			ExcludeTenantsRegex:             []string{"^Team C$"},
			LookUpDefaultWorkerPools:        false,
			ExcludeRunbooksRegex:            []string{"^MyRunbook$"},
			ExcludeRunbooks:                 []string{"MyRunbook2"},
			ExcludeLibraryVariableSetsRegex: []string{"^Test2$"},
			ExcludeLibraryVariableSets:      []string{"Test3"},
			ExcludeProjectVariablesRegex:    []string{"Excluded.*"},
			ExcludeProjectVariables:         []string{"NamedExcluded"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 2 {
					return errors.New("There must only be two projects")
				}

				testProject := lo.Filter(projectCollection.Items, func(item octopus.Project, index int) bool {
					return item.Name == "Test"
				})

				if len(testProject) != 1 {
					return errors.New("The project must be called \"Test\"")
				}

				if len(testProject[0].IncludedLibraryVariableSetIds) != 1 {
					return errors.New("The project must link to only 1 variable set (as the others were excluded)")
				}

				// Verify that the variable set was imported

				if testProject[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *testProject[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				if len(variableSet.Variables) != 6 {
					return errors.New("The project must have 6 variables")
				}

				if !lo.SomeBy(variableSet.Variables, func(item octopus.Variable) bool {
					return item.Name == "Test"
				}) {
					return errors.New("The project must have 1 variable called \"Test\"")
				}

				// The following tests ensure that variables referencing resources like feeds, accounts, worker pools,
				// and git credentials were correctly reassigned to the appropriate values in the new space.

				// Ensure a variable that referenced a feed was correctly recreated
				feedVar := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "HelmFeed"
				})

				if len(feedVar) != 1 {
					return errors.New("The project must have 1 variable called \"HelmFeed\"")
				}

				feed := octopus.Feed{}
				_, err = octopusClient.GetSpaceResourceById("Feeds", strutil.EmptyIfNil(feedVar[0].Value), &feed)

				if err != nil {
					return err
				}

				if strutil.EmptyIfNil(feed.FeedType) != "Helm" {
					return errors.New("The project must reference the helm feed as a variable")
				}

				if len(testProject[0].IncludedLibraryVariableSetIds) != 1 {
					return errors.New("The project must link to only 1 variable set (as the others were excluded)")
				}

				// Ensure a variable that referenced an account was correctly recreated
				accountVar := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "AwsAccount"
				})

				if len(accountVar) != 1 {
					return errors.New("The project must have 1 variable called \"AwsAccount\"")
				}

				account := octopus.Account{}
				_, err = octopusClient.GetSpaceResourceById("Accounts", strutil.EmptyIfNil(accountVar[0].Value), &account)

				if err != nil {
					return err
				}

				if account.AccountType != "AmazonWebServicesAccount" {
					return errors.New("The project must reference the aws account as a variable")
				}

				// Ensure a variable that referenced wokrer pools was correctly recreated
				workerPoolVar := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "WorkerPool"
				})

				if len(workerPoolVar) != 1 {
					return errors.New("The project must have 1 variable called \"WorkerPool\"")
				}

				workerPool := octopus.WorkerPool{}
				_, err = octopusClient.GetSpaceResourceById("WorkerPools", strutil.EmptyIfNil(workerPoolVar[0].Value), &workerPool)

				if err != nil {
					return err
				}

				if workerPool.Name != "Default Worker Pool" {
					return errors.New("The project must reference the default worker pool as a variable")
				}

				// Ensure a variable that referenced certificates was correctly recreated
				certificateVar := lo.Filter(variableSet.Variables, func(item octopus.Variable, index int) bool {
					return item.Name == "Certificate"
				})

				if len(certificateVar) != 1 {
					return errors.New("The project must have 1 variable called \"Certificate\"")
				}

				certificate := octopus.Certificate{}
				_, err = octopusClient.GetSpaceResourceById("Certificates", strutil.EmptyIfNil(certificateVar[0].Value), &certificate)

				if err != nil {
					return err
				}

				if certificate.Name != "Test" {
					return errors.New("The project must reference the certificate called \"Test\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			err = func() error {
				runbookCollection := octopus.GeneralCollection[octopus.Runbook]{}
				err := octopusClient.GetAllResources("Runbooks", &runbookCollection)

				if err != nil {
					return err
				}

				runbook := lo.Filter(runbookCollection.Items, func(item octopus.Runbook, index int) bool {
					return item.Name == "MyRunbook3"
				})

				if len(runbook) != 1 {
					return errors.New("Should have created a runbook called \"MyRunbook3\"")
				}

				runbookProcess := octopus.RunbookProcess{}
				_, err = octopusClient.GetSpaceResourceById("RunbookProcesses", strutil.EmptyIfNil(runbook[0].RunbookProcessId), &runbookProcess)

				if err != nil {
					return err
				}

				if strutil.EmptyIfNil(runbookProcess.Steps[0].Actions[0].Packages[0].FeedId) != "#{HelmFeed}" {
					return errors.New("Package feed should have been \"#{HelmFeed}\" (was" + strutil.EmptyIfNil(runbookProcess.Steps[0].Actions[0].Packages[0].FeedId) + " )")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			err = func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				project := lo.Filter(projectCollection.Items, func(item octopus.Project, index int) bool {
					return item.Name == "Lookup project"
				})

				if len(project) != 1 {
					return errors.New("Should have created a project called \"Lookup project\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single channel was exported
			err = func() error {
				channelsCollection := octopus.GeneralCollection[octopus.Channel]{}
				err = octopusClient.GetAllResources("Channels", &channelsCollection)

				if err != nil {
					return err
				}

				foundChannel := false
				for _, v := range channelsCollection.Items {
					if v.Name == "Test 1" {
						foundChannel = true
					}
				}

				if !foundChannel {
					return errors.New("The space must have a channel called \"Test 1\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single library variable sets were exported
			err = func() error {
				collection := octopus.GeneralCollection[octopus.LibraryVariableSet]{}
				err = octopusClient.GetAllResources("LibraryVariableSets", &collection)

				if err != nil {
					return err
				}

				if len(lo.Filter(collection.Items, func(item octopus.LibraryVariableSet, index int) bool { return item.Name == "Test" })) != 1 {
					return errors.New("The space must have a library variable set called \"Test\"")
				}

				if len(lo.Filter(collection.Items, func(item octopus.LibraryVariableSet, index int) bool { return item.Name == "Test2" })) != 1 {
					return errors.New("The space must have a library variable set called \"Test2\"")
				}

				if len(lo.Filter(collection.Items, func(item octopus.LibraryVariableSet, index int) bool { return item.Name == "Test3" })) != 1 {
					return errors.New("The space must have a library variable set called \"Test3\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the single trigger was exported
			err = func() error {
				triggersCollection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
				err = octopusClient.GetAllResources("ProjectTriggers", &triggersCollection)

				if err != nil {
					return err
				}

				foundTrigger := false
				for _, v := range triggersCollection.Items {
					if v.Name == "Test 1" {
						foundTrigger = true
					}
				}

				if !foundTrigger {
					return errors.New("The space must have a trigger called \"Test 1\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			// Verify that the runbook was excluded
			err = func() error {
				collection := octopus.GeneralCollection[octopus.Runbook]{}
				err := octopusClient.GetAllResources("Runbooks", &collection)

				if err != nil {
					return err
				}

				if len(collection.Items) != 1 {
					return errors.New("One runbook should have been exported")
				}

				if collection.Items[0].Name != "MyRunbook3" {
					return errors.New("The runbook should be called MyRunbook3")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectLookupExportWithWorkerPool verifies that a single project can be reimported with the correct worker pool.
func TestSingleProjectLookupExportWithWorkerPool(t *testing.T) {
	if os.Getenv("GIT_CREDENTIAL") == "" {
		t.Fatalf("the GIT_CREDENTIAL environment variable must be set to a GitHub access key")
	}

	exportProjectLookupImportAndTest(
		t,
		"Test 2",
		"../test/terraform/43-multipleprojectslookup/space_creation",
		"../test/terraform/43-multipleprojectslookup/space_prepopulation",
		"../test/terraform/43-multipleprojectslookup/space_population",
		"../test/terraform/43-multipleprojectslookup/space_creation",
		"../test/terraform/43-multipleprojectslookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{
			"-var=gitcredential_matt=" + os.Getenv("GIT_CREDENTIAL"),
		},
		[]string{},
		args2.Arguments{
			ExcludeAllTenants: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				testProject := lo.Filter(projectCollection.Items, func(item octopus.Project, index int) bool {
					return item.Name == "Test 2"
				})

				if len(testProject) != 1 {
					return errors.New("The project must be called \"Test 2\"")
				}

				workerPoolCollection := octopus.GeneralCollection[octopus.WorkerPool]{}
				err = octopusClient.GetAllResources("WorkerPools", &workerPoolCollection)

				if err != nil {
					return err
				}

				dockerWorkerPool := lo.Filter(workerPoolCollection.Items, func(item octopus.WorkerPool, index int) bool {
					return item.Name == "Docker"
				})

				if len(dockerWorkerPool) != 1 {
					return errors.New("Should have created a worker pool called \"Docker\"")
				}

				deploymentProcess := octopus.DeploymentProcess{}
				found, err := octopusClient.GetSpaceResourceById("DeploymentProcesses",
					strutil.EmptyIfNil(testProject[0].DeploymentProcessId),
					&deploymentProcess)

				if err != nil {
					return err
				}

				if !found {
					return errors.New("Expected to find a deployment process")
				}

				if deploymentProcess.Steps[0].Actions[0].WorkerPoolId != dockerWorkerPool[0].Id {
					return errors.New("Action should have worker pool set to Docker (was" + deploymentProcess.Steps[0].Actions[0].WorkerPoolId + " )")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestProjectWithGitUsernameExport verifies that a project can be reimported with the correct git settings
func TestProjectWithGitUsernameExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/39-projectgitusername/space_creation",
		"../test/terraform/39-projectgitusername/space_population",
		[]string{
			"-var=project_git_password=" + os.Getenv("GIT_CREDENTIAL"),
		},
		[]string{
			"-var=project_test_git_password=" + os.Getenv("GIT_CREDENTIAL"),
			"-var=project_test_git_base_path=.octopus/projectgitusername",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if v.PersistenceSettings.Credentials.Type != "UsernamePassword" {
						return errors.New("The project must be have a git credential type of \"UsernamePassword\" (was \"" + v.PersistenceSettings.Credentials.Type + "\")")
					}

					if v.PersistenceSettings.Credentials.Username != "mcasperson" {
						return errors.New("The project must be have a git username of \"mcasperson\" (was \"" + v.PersistenceSettings.Credentials.Username + "\")")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectWithDollarSignsExport verifies that a project can be reimported with terraform string interpolation
func TestProjectWithDollarSignsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/40-escapedollar/space_creation",
		"../test/terraform/40-escapedollar/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectTerraformInlineScriptExport verifies that a project can be reimported with a terraform inline template step
func TestProjectTerraformInlineScriptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/41-terraforminlinescript/space_creation",
		"../test/terraform/41-terraforminlinescript/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestGithubFeedExport verifies that a github feed can be reimported with the correct settings
func TestGithubFeedExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/42-githubfeed/space_creation",
		"../test/terraform/42-githubfeed/space_population",
		[]string{},
		[]string{
			"-var=feed_github_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Feed]{}
			err := octopusClient.GetAllResources("Feeds", &collection)

			if err != nil {
				return err
			}

			feedName := "Github"
			found := false
			for _, v := range collection.Items {
				if v.Name == feedName {
					found = true

					if strutil.EmptyIfNil(v.FeedType) != "GitHub" {
						return errors.New("The feed must have a type of \"GitHub\", was \"" + strutil.EmptyIfNil(v.FeedType) + "\"")
					}

					if strutil.EmptyIfNil(v.Username) != "test-username" {
						return errors.New("The feed must have a username of \"test-username\", was \"" + strutil.EmptyIfNil(v.Username) + "\"")
					}

					if !v.Password.HasValue {
						return errors.New("The feed must have a password")
					}

					if intutil.ZeroIfNil(v.DownloadAttempts) != 1 {
						return errors.New("The feed must be have a downloads attempts set to \"1\"")
					}

					if intutil.ZeroIfNil(v.DownloadRetryBackoffSeconds) != 30 {
						return errors.New("The feed must be have a downloads retry backoff set to \"30\"")
					}

					if strutil.EmptyIfNil(v.FeedUri) != "https://api.github.com" {
						return errors.New("The feed must be have a feed uri of \"https://api.github.com\", was \"" + strutil.EmptyIfNil(v.FeedUri) + "\"")
					}

					foundExecutionTarget := false
					foundServer := false
					for _, o := range v.PackageAcquisitionLocationOptions {
						if o == "ExecutionTarget" {
							foundExecutionTarget = true
						}

						if o == "Server" {
							foundServer = true
						}
					}

					if !(foundExecutionTarget && foundServer) {
						return errors.New("The feed must be have a PackageAcquisitionLocationOptions including \"ExecutionTarget\" and \"Server\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an feed called \"" + feedName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestRunbookExport verifies that a runbook can be reimported with the correct settings
func TestRunbookExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/44-runbooks/space_creation",
		"../test/terraform/44-runbooks/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeRunbooksExcept: []string{"Runbook"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Runbook]{}
			err := octopusClient.GetAllResources("Runbooks", &collection)

			if err != nil {
				return err
			}

			resourceName := "Runbook"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test Runbook" {
						return errors.New("The runbook must have a description of \"GitHub\", was \"" + strutil.EmptyIfNil(v.Description) + "\"")
					}

					if strutil.EmptyIfNil(v.MultiTenancyMode) != "Untenanted" {
						return errors.New("The runbook must have a MultiTenancyMode of \"Untenanted\", was \"" + strutil.EmptyIfNil(v.MultiTenancyMode) + "\"")
					}

					if strutil.EmptyIfNil(v.EnvironmentScope) != "Specified" {
						return errors.New("The runbook must have a EnvironmentScope of \"Specified\", was \"" + strutil.EmptyIfNil(v.EnvironmentScope) + "\"")
					}

					if strutil.EmptyIfNil(v.DefaultGuidedFailureMode) != "EnvironmentDefault" {
						return errors.New("The runbook must have a DefaultGuidedFailureMode of \"EnvironmentDefault\", was \"" + strutil.EmptyIfNil(v.DefaultGuidedFailureMode) + "\"")
					}

					if !v.ForcePackageDownload {
						t.Log("BUG: The runbook must have a ForcePackageDownload of \"true\", was \"" + fmt.Sprint(v.ForcePackageDownload) + "\"")
					}

					if len(v.Environments) != 1 {
						return errors.New("The runbook must have a 1 Environments, was \"" + fmt.Sprint(len(v.Environments)) + "\"")
					}

					if v.ConnectivityPolicy.SkipMachineBehavior != "SkipUnavailableMachines" {
						return errors.New("The runbook must have a ConnectivityPolicy.SkipMachineBehavior of \"SkipUnavailableMachines\", was \"" + v.ConnectivityPolicy.SkipMachineBehavior + "\"")
					}

					if v.ConnectivityPolicy.AllowDeploymentsToNoTargets {
						return errors.New("The runbook must have a ConnectivityPolicy.AllowDeploymentsToNoTargets of \"false\", was \"" + fmt.Sprint(v.ConnectivityPolicy.AllowDeploymentsToNoTargets) + "\"")
					}

					if v.ConnectivityPolicy.ExcludeUnhealthyTargets {
						return errors.New("The runbook must have a ConnectivityPolicy.ExcludeUnhealthyTargets of \"false\", was \"" + fmt.Sprint(v.ConnectivityPolicy.ExcludeUnhealthyTargets) + "\"")
					}

					process := octopus.RunbookProcess{}
					_, err := octopusClient.GetSpaceResourceById("RunbookProcesses", strutil.EmptyIfNil(v.RunbookProcessId), &process)

					if err != nil {
						return errors.New("Failed to retrieve the runbook process")
					}

					if len(process.Steps) != 2 {
						return errors.New("The runbook must have a 2 steps, was \"" + fmt.Sprint(len(process.Steps)) + "\"")
					}

					if strutil.EmptyIfNil(process.Steps[0].Name) != "Hello world (using PowerShell)" {
						return errors.New("The runbook step must have a name of \"Hello world (using PowerShell)\", was \"" + strutil.EmptyIfNil(process.Steps[0].Name) + "\"")
					}

					if fmt.Sprint(process.Steps[0].Actions[0].Properties["Octopus.Action.EnabledFeatures"]) != "Octopus.Features.JsonConfigurationVariables" {
						return errors.New("The runbook step must have the feature \"Octopus.Features.JsonConfigurationVariables\" enabled (was \"" + fmt.Sprint(process.Steps[0].Actions[0].Properties["Octopus.Action.EnabledFeatures"]) + "\"")
					}

					if len(process.Steps[0].Actions[0].Packages) != 1 {
						return errors.New("The runbook must have one package")
					}

					if strutil.EmptyIfNil(process.Steps[0].Actions[0].Packages[0].Name) != "package1" {
						return errors.New("The runbook must have one package called \"package1\", was \"" + strutil.EmptyIfNil(process.Steps[0].Actions[0].Packages[0].Name) + "\"")
					}

					if strutil.EmptyIfNil(process.Steps[1].Name) != "Test" {
						return errors.New("The runbook step must have a name of \"Test\", was \"" + strutil.EmptyIfNil(process.Steps[1].Name) + "\"")
					}

					if len(process.Steps[1].Actions[0].Packages) != 1 {
						return errors.New("The runbook must have one package")
					}

					if strutil.EmptyIfNil(process.Steps[1].Actions[0].Packages[0].Name) != "" {
						return errors.New("The runbook must have one unnamed primary package, was \"" + strutil.EmptyIfNil(process.Steps[1].Actions[0].Packages[0].Name) + "\"")
					}
				}
			}

			if !found {
				return errors.New("Space must have an runbook called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestRunbookExcludeExceptExport verifies that a runbook can be excluded
func TestRunbookExcludeExceptExport(t *testing.T) {
	exportProjectImportAndTest(t,
		"Test",
		"../test/terraform/44-runbooks/space_creation",
		"../test/terraform/44-runbooks/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeRunbooksExcept: []string{"DoesNotExist"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Runbook]{}
			err := octopusClient.GetAllResources("Runbooks", &collection)

			if err != nil {
				return err
			}

			if len(collection.Items) != 0 {
				return errors.New("Space must not have any runbooks in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestK8sTargetWithCertExport verifies that a k8s machine with cert auth can be reimported with the correct settings
func TestK8sTargetWithCertExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/45-k8scertauth/space_creation",
		"../test/terraform/45-k8scertauth/space_population",
		[]string{},
		[]string{
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesCertificate" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"KubernetesCertificate\" (was \"" + machine.Endpoint.Authentication.AuthenticationType + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSingleProjectWithAccountLookupExport verifies that a single project referencing an account can be reimported with the correct settings.
func TestSingleProjectWithAccountLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/46-awsaccountprojectlookup/space_creation",
		"../test/terraform/46-awsaccountprojectlookup/space_prepopulation",
		"../test/terraform/46-awsaccountprojectlookup/space_population",
		"../test/terraform/46-awsaccountprojectlookup/space_creation",
		"../test/terraform/46-awsaccountprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				awsVars := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "AWS" })

				if len(awsVars) == 0 {
					return errors.New("The project must have 1 variable called \"AWS\"")
				}

				if len(awsVars[0].Scope.Environment) != 1 {
					return errors.New("The project must have 1 variable called \"AWS\" scoped to an environment")
				}

				azureVars := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "Azure" })

				if len(azureVars) == 0 {
					return errors.New("The project must have 1 variable called \"Azure\"")
				}

				if len(azureVars[0].Scope.Environment) != 1 {
					return errors.New("The project must have 1 variable called \"Azure\" scoped to an environment")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithMachineScopedVarLookupExport verifies that a single project with a variable scoped to a bunch of machines can be reimported with lookups using correct settings.
func TestSingleProjectWithMachineScopedVarLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/47-targetprojectlookup/space_creation",
		"../test/terraform/47-targetprojectlookup/space_prepopulation",
		"../test/terraform/47-targetprojectlookup/space_population",
		"../test/terraform/47-targetprojectlookup/space_creation",
		"../test/terraform/47-targetprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				machineCollection := octopus.GeneralCollection[octopus.Machine]{}
				err = octopusClient.GetAllResources("Machines", &machineCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project (found " + fmt.Sprint(len(projectCollection.Items)) + ")")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if len(scopedVar[0].Scope.Machine) != 9 {
					machineTypes := lo.Map(machineCollection.Items, func(item octopus.Machine, index int) string {
						return item.Id + " " + item.Endpoint.CommunicationStyle
					})

					return errors.New("The project must have 1 variable called \"test\" scoped to nine machines " +
						"(was scoped to " + fmt.Sprint(len(scopedVar[0].Scope.Machine)) + " machines). " +
						"The following machines were available in the space: " + strings.Join(machineTypes, ", "))
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithMachineScopedVarExport verifies that a single project with a variable scoped to a bunch of
// machines can be reimported with the correct settings.
func TestSingleProjectWithMachineScopedVarExport(t *testing.T) {
	exportProjectImportAndTest(t,
		"Test",
		"../test/terraform/50-targetproject/space_creation",
		"../test/terraform/50-targetproject/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=target_servicefabric=blah",
			"-var=account_sales_account=blah",
			"-var=account_ec2_sydney_cert=blah",
			"-var=account_ec2_sydney=blah",
			"-var=account_aws_account=blah",
			"-var=account_subscription_cert=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",
			"-var=certificate_test_data=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",
			"-var=certificate_test_password=Password01!",
		},
		args2.Arguments{
			ExcludeProjectVariables: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				machineCollection := octopus.GeneralCollection[octopus.Machine]{}
				err = octopusClient.GetAllResources("Machines", &machineCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project (found " + fmt.Sprint(len(projectCollection.Items)) + ")")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if len(scopedVar[0].Scope.Machine) != 9 {
					machineTypes := lo.Map(machineCollection.Items, func(item octopus.Machine, index int) string {
						return item.Endpoint.CommunicationStyle
					})

					return errors.New("The project must have 1 variable called \"test\" scoped to nine machines " +
						"(was scoped to " + fmt.Sprint(len(scopedVar[0].Scope.Machine)) + " machines). " +
						"The following machines were available in the space: " + strings.Join(machineTypes, ", "))
				}

				// Validate the k8s target
				collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
				err = octopusClient.GetAllResources("Machines", &collection)

				if err != nil {
					return err
				}

				resourceName := "Test"
				foundK8sTarget := false

				for _, machine := range collection.Items {
					if machine.Name == resourceName {
						foundK8sTarget = true

						if strutil.EmptyIfNil(machine.Endpoint.ClusterCertificate) == "" {
							return errors.New("The k8s machine must have a cluster certificate")
						}
					}
				}

				if !foundK8sTarget {
					return errors.New("Must have imported a k8s target called \"Test\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithExcludedMachineScopedVarExport verifies that a single project with a variable scoped to a bunch of
// machines, that are then excluded, can be reimported with the correct settings.
func TestSingleProjectWithExcludedMachineScopedVarExport(t *testing.T) {
	exportProjectImportAndTest(t,
		"Test",

		"../test/terraform/50-targetproject/space_creation",
		"../test/terraform/50-targetproject/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllTargets:       true,
			ExcludeProjectVariables: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if len(scopedVar[0].Scope.Machine) != 0 {
					return errors.New("The project must have 1 variable called \"test\" scoped to no machines because they were excluded")
				}

				collection := octopus.GeneralCollection[octopus.NameId]{}
				err = octopusClient.GetAllResources("Machines", &collection)

				if err != nil {
					return err
				}

				if len(collection.Items) != 0 {
					return errors.New("No machines should have been exported, but found " + strings.Join(lo.Map(collection.Items, func(item octopus.NameId, index int) string {
						return item.Name
					}), ", "))
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithCertificateVarLookupExport verifies that a single project with a certificate variable can be reimported with the correct settings.
func TestSingleProjectWithCertificateVarLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/48-certificateprojectlookup/space_creation",
		"../test/terraform/48-certificateprojectlookup/space_prepopulation",
		"../test/terraform/48-certificateprojectlookup/space_population",
		"../test/terraform/48-certificateprojectlookup/space_creation",
		"../test/terraform/48-certificateprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if scopedVar[0].Type != "Certificate" {
					return errors.New("The project must have 1 variable called \"test\" of type certificate")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithFeedLookupExport verifies that a single project referencing multiple feeds variable can be reimported with the correct settings.
func TestSingleProjectWithFeedLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/49-feedprojectlookup/space_creation",
		"../test/terraform/49-feedprojectlookup/space_prepopulation",
		"../test/terraform/49-feedprojectlookup/space_population",
		"../test/terraform/49-feedprojectlookup/space_creation",
		"../test/terraform/49-feedprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].DeploymentProcessId == nil {
					return errors.New("The project must have a deployment process set")
				}

				deploymentProcess := octopus.DeploymentProcess{}
				_, err = octopusClient.GetSpaceResourceById("DeploymentProcesses", *projectCollection.Items[0].DeploymentProcessId, &deploymentProcess)

				if err != nil {
					return err
				}

				if len(deploymentProcess.Steps) != 1 {
					return errors.New("The project must have a deployment process with one step")
				}

				if len(deploymentProcess.Steps[0].Actions) != 1 {
					return errors.New("The project must have a deployment process with one step and one action")
				}

				if len(deploymentProcess.Steps[0].Actions[0].Packages) != 5 {
					return errors.New("The project must have a step with 5 packages")
				}

				for _, stepPackage := range deploymentProcess.Steps[0].Actions[0].Packages {
					feed := octopus.Feed{}
					_, err = octopusClient.GetSpaceResourceById("Feeds", *stepPackage.FeedId, &feed)

					if err != nil {
						return err
					}

					if strutil.EmptyIfNil(stepPackage.PackageId) == "package1" && strutil.EmptyIfNil(feed.FeedType) != "Docker" {
						return errors.New("package1 must come from a Docker feed")
					}

					if strutil.EmptyIfNil(stepPackage.PackageId) == "package2" && strutil.EmptyIfNil(feed.FeedType) != "Helm" {
						return errors.New("package2 must come from a Helm feed")
					}

					if strutil.EmptyIfNil(stepPackage.PackageId) == "package3" && strutil.EmptyIfNil(feed.FeedType) != "Maven" {
						return errors.New("package3 must come from a Maven feed")
					}

					if strutil.EmptyIfNil(stepPackage.PackageId) == "package4" && strutil.EmptyIfNil(feed.FeedType) != "NuGet" {
						return errors.New("package4 must come from a NuGet feed")
					}

					if strutil.EmptyIfNil(stepPackage.PackageId) == "package5" && strutil.EmptyIfNil(feed.FeedType) != "GitHub" {
						return errors.New("package5 must come from a GitHub feed")
					}
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestDefaultWorkerPoolExplicitLookup verifies that a deployment process using the default worker pool (i.e. with a
// blank worker pool ID) must have exported the default pool rather than reusing whatever the default pool was in the
// new space because lookUpDefaultWorkerPools was true.
func TestDefaultWorkerPoolExplicitLookup(t *testing.T) {
	exportProjectImportAndTest(t,
		"Test",
		"../test/terraform/51-defaultworkerpool/space_creation",
		"../test/terraform/51-defaultworkerpool/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			LookUpDefaultWorkerPools: true,
			ExcludeProjectVariables:  []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				deploymentProcess := octopus.DeploymentProcess{}
				_, err = octopusClient.GetSpaceResourceById("DeploymentProcesses", *projectCollection.Items[0].DeploymentProcessId, &deploymentProcess)

				if err != nil {
					return err
				}

				if len(deploymentProcess.Steps) != 1 {
					return errors.New("The project must has a step")
				}

				if len(deploymentProcess.Steps[0].Actions) != 1 {
					return errors.New("The project step must have an action")
				}

				workerPool := octopus.WorkerPool{}
				_, err = octopusClient.GetSpaceResourceById("WorkerPools", deploymentProcess.Steps[0].Actions[0].WorkerPoolId, &workerPool)

				if err != nil {
					return err
				}

				if workerPool.Name != "Docker" {
					return errors.New("Imported project must have explicitly imported the default worker pool")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestTenantCommonVarsExport verifies that a tenant with common variables can be reimported with the correct settings
func TestTenantCommonVarsExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/56-tenantcommonvars/space_creation",
		"../test/terraform/56-tenantcommonvars/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"tTest tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 2 {
						return errors.New("The tenant must have two tags")
					}

					if len(v.ProjectEnvironments) != 1 {
						return errors.New("The tenant must have one project environment, (was " + strconv.Itoa(len(v.ProjectEnvironments)) + ")")
					}

					for _, u := range v.ProjectEnvironments {
						if len(u) != 3 {
							return errors.New("The tenant must have be linked to three environments")
						}
					}

					resource := octopus.TenantVariable{}
					err = octopusClient.GetAllResources("Tenants/"+v.Id+"/Variables", &resource)

					if err != nil {
						return err
					}

					foundCount := 0

					for _, v := range resource.LibraryVariables {
						for variableId, variableValue := range v.Variables {
							template := lo.Filter(v.Templates, func(item octopus.Template, index int) bool {
								return item.Id == variableId
							})

							if len(template) != 1 {
								return errors.New("Expected to find one template that matched the variable value")
							}

							variableValueString, ok := variableValue.(string)

							if ok {
								if strutil.EmptyIfNil(template[0].Name) == "VariableA" {
									foundCount++
									if variableValueString != "Override Variable A" {
										return errors.New("Tenant variable must be Override Variable A (was " + variableValueString + ")")
									}
								}

								if strutil.EmptyIfNil(template[0].Name) == "VariableB" {
									foundCount++
									if variableValueString != "Override Variable B" {
										return errors.New("Tenant variable must be Override Variable B (was " + variableValueString + ")")
									}
								}
							}
						}
					}

					if foundCount != 2 {
						return errors.New("Should have found two regular variables")
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSingleProjectWithScriptModuleLookupExport verifies that a single project referencing a script module can be reimported with the correct settings.
func TestSingleProjectWithScriptModuleLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/57-scriptmoduleprojectlookup/space_creation",
		"../test/terraform/57-scriptmoduleprojectlookup/space_prepopulation",
		"../test/terraform/57-scriptmoduleprojectlookup/space_population",
		"../test/terraform/57-scriptmoduleprojectlookup/space_creation",
		"../test/terraform/57-scriptmoduleprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				if len(projectCollection.Items[0].IncludedLibraryVariableSetIds) != 1 {
					return errors.New("The project must link to 1 library variable set")
				}

				resource := octopus.LibraryVariableSet{}
				_, err = octopusClient.GetSpaceResourceById("LibraryVariableSets", projectCollection.Items[0].IncludedLibraryVariableSetIds[0], &resource)

				if err != nil {
					return err
				}

				if resource.Name != "Script Module" {
					return errors.New("The project must link to 1 library variable set called \"Script Module\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestProjectWithScriptModuleExport verifies that a project with a script module can be reimported with the correct settings
func TestProjectWithScriptModuleExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/58-scriptmodule/space_creation",
		"../test/terraform/58-scriptmodule/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if len(v.IncludedLibraryVariableSetIds) != 1 {
						return errors.New("The project must have a library variable set")
					}

					resource := octopus.LibraryVariableSet{}
					_, err = octopusClient.GetSpaceResourceById("LibraryVariableSets", v.IncludedLibraryVariableSetIds[0], &resource)

					if err != nil {
						return err
					}

					if resource.Name != "Script Module" {
						return errors.New("The project must link to 1 library variable set called \"Script Module\"")
					}

				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestSingleProjectWithTagsetScopedVarLookupExport verifies that a single project with a variable scoped to a tagset can be reimported with lookups using correct settings.
func TestSingleProjectWithTagsetScopedVarLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/59-tagsetprojectlookup/space_creation",
		"../test/terraform/59-tagsetprojectlookup/space_prepopulation",
		"../test/terraform/59-tagsetprojectlookup/space_population",
		"../test/terraform/59-tagsetprojectlookup/space_creation",
		"../test/terraform/59-tagsetprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if len(scopedVar[0].Scope.TenantTag) != 1 {
					return errors.New("The project must have 1 variable called \"test\" scoped to 1 tagset")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestSingleProjectWithTenantedMachineScopedVarLookupExport verifies that a single project with a variable scoped to a bunch of machines can be reimported with lookups using correct settings.
func TestSingleProjectWithTenantedMachineScopedVarLookupExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/60-tenantprojectlookup/space_creation",
		"../test/terraform/60-tenantprojectlookup/space_prepopulation",
		"../test/terraform/60-tenantprojectlookup/space_population",
		"../test/terraform/60-tenantprojectlookup/space_creation",
		"../test/terraform/60-tenantprojectlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				if len(scopedVar[0].Scope.Machine) != 1 {
					return errors.New("The project must have 1 variable called \"test\" scoped to 1 machines")
				}

				machine := octopus.CloudRegionResource{}
				_, err = octopusClient.GetSpaceResourceById("Machines", scopedVar[0].Scope.Machine[0], &machine)

				if err != nil {
					return err
				}

				if machine.Name != "CloudRegion" {
					return errors.New("The machine called CloudRegion must have been exported")
				}

				if len(machine.TenantIds) != 1 {
					return errors.New("The machine called CloudRegion needs to be assigned to 1 tenant")
				}

				tenant := octopus.Tenant{}
				_, err = octopusClient.GetSpaceResourceById("Tenants", machine.TenantIds[0], &tenant)

				if tenant.Name != "Team A" {
					return errors.New("The tenant called Team A must have been exported")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestTenantsWithExcludedProjectExport verifies that a tenant can be reimported with the correct settings after a project is excluded
func TestTenantsWithExcludedProjectExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/61-tenantswithexcludedprojects/space_creation",
		"../test/terraform/61-tenantswithexcludedprojects/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeProjects: []string{"Test2"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"Test tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 2 {
						return errors.New("The tenant must have two tags")
					}

					if len(v.ProjectEnvironments) != 1 {
						return errors.New("The tenant must have one project environment, (was " + strconv.Itoa(len(v.ProjectEnvironments)) + ")")
					}

					for _, u := range v.ProjectEnvironments {
						if len(u) != 3 {
							return errors.New("The tenant must have be linked to three environments")
						}
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			projects := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projects)

			if err != nil {
				return err
			}

			if len(projects.Items) != 1 {
				return errors.New("Only one project should have been exported, as the second was excluded.")
			}

			return nil
		})
}

// TestTenantsWithExcludedAllProjectExport verifies that a tenant can be reimported with the correct settings after all projects are excluded
func TestTenantsWithExcludedAllProjectExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/61-tenantswithexcludedprojects/space_creation",
		"../test/terraform/61-tenantswithexcludedprojects/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeAllProjects: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"Test tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 2 {
						return errors.New("The tenant must have two tags")
					}

					if len(v.ProjectEnvironments) != 0 {
						return errors.New("The tenant must have zero project environments")
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			projects := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projects)

			if err != nil {
				return err
			}

			if len(projects.Items) != 0 {
				return errors.New("No projects should have been exported, as they were all excluded.")
			}

			return nil
		})
}

// TestTenantsWithExcludedProjectRegexExport verifies that a tenant can be reimported with the correct settings after a project is excluded
func TestTenantsWithExcludedProjectRegexExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/61-tenantswithexcludedprojects/space_creation",
		"../test/terraform/61-tenantswithexcludedprojects/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeProjectsRegex: []string{"^Test2$"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"Test tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 2 {
						return errors.New("The tenant must have two tags")
					}

					if len(v.ProjectEnvironments) != 1 {
						return errors.New("The tenant must have one project environment, (was " + strconv.Itoa(len(v.ProjectEnvironments)) + ")")
					}

					for _, u := range v.ProjectEnvironments {
						if len(u) != 3 {
							return errors.New("The tenant must have be linked to three environments")
						}
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			projects := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projects)

			if err != nil {
				return err
			}

			if len(projects.Items) != 1 {
				return errors.New("Only one project should have been exported, as the second was excluded.")
			}

			return nil
		})
}

// TestTenantsWithExcludedProjectExceptExport verifies that a tenant can be reimported with the correct settings after a project is excluded
func TestTenantsWithExcludedProjectExceptExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/61-tenantswithexcludedprojects/space_creation",
		"../test/terraform/61-tenantswithexcludedprojects/space_population",
		[]string{},
		[]string{},
		args2.Arguments{
			ExcludeProjectsExcept: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Tenant]{}
			err := octopusClient.GetAllResources("Tenants", &collection)

			if err != nil {
				return err
			}

			resourceName := "Team A"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test tenant" {
						return errors.New("The tenant must be have a description of \"Test tenant\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}

					if len(v.TenantTags) != 2 {
						return errors.New("The tenant must have two tags")
					}

					if len(v.ProjectEnvironments) != 1 {
						return errors.New("The tenant must have one project environment, (was " + strconv.Itoa(len(v.ProjectEnvironments)) + ")")
					}

					for _, u := range v.ProjectEnvironments {
						if len(u) != 3 {
							return errors.New("The tenant must have be linked to three environments")
						}
					}
				}
			}

			if !found {
				return errors.New("Space must have an tenant called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			projects := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projects)

			if err != nil {
				return err
			}

			if len(projects.Items) != 1 {
				return errors.New("Only one project should have been exported, as the second was excluded.")
			}

			return nil
		})
}

// TestSingleProjectWithExcludedMachineScopedVarExport verifies that a single project with a variable referencing an
// account scoped to a tenant can be reimported with the correct settings.
func TestSingleProjectWithAccountAndTenantExport(t *testing.T) {
	exportProjectImportAndTest(t,
		"Test",
		"../test/terraform/62-tokenaccountwithtenant/space_creation",
		"../test/terraform/62-tokenaccountwithtenant/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=account_token=blah",
		},
		args2.Arguments{
			ExcludeAllTargets:       true,
			ExcludeProjectVariables: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				projectCollection := octopus.GeneralCollection[octopus.Project]{}
				err := octopusClient.GetAllResources("Projects", &projectCollection)

				if err != nil {
					return err
				}

				if len(projectCollection.Items) != 1 {
					return errors.New("There must only be one project")
				}

				if projectCollection.Items[0].Name != "Test" {
					return errors.New("The project must be called \"Test\"")
				}

				// Verify that the variable set was imported

				if projectCollection.Items[0].VariableSetId == nil {
					return errors.New("The project must have a variable set")
				}

				variableSet := octopus.VariableSet{}
				_, err = octopusClient.GetSpaceResourceById("Variables", *projectCollection.Items[0].VariableSetId, &variableSet)

				if err != nil {
					return err
				}

				scopedVar := lo.Filter(variableSet.Variables, func(x octopus.Variable, index int) bool { return x.Name == "test" })
				if len(scopedVar) == 0 {
					return errors.New("The project must have 1 variable called \"test\"")
				}

				accounts := octopus.GeneralCollection[octopus.NameId]{}
				err = octopusClient.GetAllResources("Accounts", &accounts)

				if err != nil {
					return err
				}

				if len(accounts.Items) != 1 {
					return errors.New("One account should have been imported via the account variable")
				}

				tenants := octopus.GeneralCollection[octopus.Tenant]{}
				err = octopusClient.GetAllResources("Tenants", &tenants)

				if err != nil {
					return err
				}

				if len(tenants.Items) != 1 {
					return errors.New("One tenant should have been imported via the account variable")
				}

				if strutil.EmptyIfNil(scopedVar[0].Value) != accounts.Items[0].Id {
					return errors.New("The variable should reference the new account id")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestK8sPodAuthExport verifies that a k8s target with pod authentication can be exported with the correct values.
func TestK8sPodAuthExport(t *testing.T) {
	exportSpaceImportAndTest(t,
		"../test/terraform/63-k8stargetpodauth/space_creation",
		"../test/terraform/63-k8stargetpodauth/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.KubernetesEndpointResource]{}
			err := octopusClient.GetAllResources("Machines", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			foundResource := false

			for _, machine := range collection.Items {
				if machine.Name == resourceName {
					foundResource = true

					if strutil.EmptyIfNil(machine.Endpoint.ClusterUrl) != "https://cluster" {
						return errors.New("The machine must have a Endpoint.ClusterUrl of \"https://cluster\" (was \"" + fmt.Sprint(machine.Endpoint.ClusterUrl) + "\")")
					}

					if machine.Endpoint.Authentication.AuthenticationType != "KubernetesPodService" {
						return errors.New("The machine must have a Endpoint.Authentication.AuthenticationType of \"KubernetesPodService\" (was \"" + fmt.Sprint(machine.Endpoint.Authentication.AuthenticationType) + "\")")
					}

					if strutil.EmptyIfNil(machine.Endpoint.Authentication.TokenPath) != "/var/run/secrets/kubernetes.io/serviceaccount/token" {
						return errors.New("The machine must have a Endpoint.Authentication.TokenPath of \"/var/run/secrets/kubernetes.io/serviceaccount/token\" (was \"" + fmt.Sprint(machine.Endpoint.Authentication.TokenPath) + "\")")
					}

					if strutil.EmptyIfNil(machine.Endpoint.ClusterCertificatePath) != "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt" {
						return errors.New("The machine must have a Endpoint.ClusterCertificatePath of \"/var/run/secrets/kubernetes.io/serviceaccount/ca.crt\" (was \"" + fmt.Sprint(machine.Endpoint.ClusterCertificatePath) + "\")")
					}
				}
			}

			if !foundResource {
				return errors.New("Space must have a target \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestTenantSensitiveVariablesExport verifies that sensitive tenant variables can be reimported with the correct settings
func TestTenantSensitiveVariablesExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/64-projectvariabletemplate/space_creation",
		"../test/terraform/64-projectvariabletemplate/space_population",
		[]string{},
		[]string{},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			project := lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
				return item.Name == "Test"
			})

			if len(project) != 1 {
				return errors.New("Must have a project called \"Test\"")
			}

			template := lo.Filter(project[0].Templates, func(item octopus.Template, index int) bool {
				if strutil.EmptyIfNil(item.Name) != "Project Template Variable" {
					return false
				}

				if item.DisplaySettings["Octopus.ControlType"] != "Sensitive" {
					return false
				}

				if strutil.EmptyIfNil(item.Label) != "Test" {
					return false
				}

				// Test as a string. This is probably a bug, but the template schema
				// https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy/latest/docs/resources/project#nestedblock--template
				// has no other way to store a value than default_value, which sets the value as a plain string.
				defaultValue, ok := item.DefaultValue.(string)
				if ok {
					if defaultValue != "replace me with a password" {
						return false
					}
				} else {
					// Otherwise inspect the default value as a secret placeholder
					r := reflect.ValueOf(item.DefaultValue)
					f := reflect.Indirect(r).FieldByName("HasValue")

					if f.CanConvert(reflect.TypeOf(true)) {
						if !f.Bool() {
							return false
						}
					}
				}

				return true
			})

			if len(template) != 1 {
				return errors.New("Must have found a sensitive template variable.")
			}

			singleLineTemplate := lo.Filter(project[0].Templates, func(item octopus.Template, index int) bool {
				return strutil.EmptyIfNil(item.Name) == "Project Template Variable 2" &&
					item.DisplaySettings["Octopus.ControlType"] == "SingleLineText" &&
					strutil.EmptyIfNil(item.Label) == "Test2" &&
					strutil.EmptyIfNil(item.GetDefaultValueString()) == "Test2"
			})

			if len(singleLineTemplate) != 1 {
				return errors.New("Must have found a single line template variable")
			}

			return nil
		})
}

// TestTenantedResources verifies that resources assigned with tenant tags can be exported and recreated.
// This test verifies that tenant tags are created in the correct order, as these are an example of Octopus resources
// linked by plain strings rather than by IDs.
func TestTenantedResources(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/67-tenanttaggedresources/space_creation",
		"../test/terraform/67-tenanttaggedresources/space_population",
		[]string{
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
			"-var=target_service_fabric=whatever",
			"-var=account_sales_account=whatever",
		},
		[]string{
			"-var=certificate_test_data=MIIQoAIBAzCCEFYGCSqGSIb3DQEHAaCCEEcEghBDMIIQPzCCBhIGCSqGSIb3DQEHBqCCBgMwggX/AgEAMIIF+AYJKoZIhvcNAQcBMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAjBMRI6S6M9JgICCAAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEFTttp7/9moU4zB8mykyT2eAggWQBGjcI6T8UT81dkN3emaXFXoBY4xfqIXQ0nGwUUAN1TQKOY2YBEGoQqsfB4yZrUgrpP4oaYBXevvJ6/wNTbS+16UOBMHu/Bmi7KsvYR4i7m2/j/SgHoWWKLmqOXgZP7sHm2EYY74J+L60mXtUmaFO4sHoULCwCJ9V3/l2U3jZHhMVaVEB0KSporDF6oO5Ae3M+g7QxmiXsWoY1wBFOB+mrmGunFa75NEGy+EyqfTDF8JqZRArBLn1cphi90K4Fce51VWlK7PiJOdkkpMVvj+mNKEC0BvyfcuvatzKuTJsnxF9jxsiZNc28rYtxODvD3DhrMkK5yDH0h9l5jfoUxg+qHmcY7TqHqWiCdExrQqUlSGFzFNInUF7YmjBRHfn+XqROvYo+LbSwEO+Q/QViaQC1nAMwZt8PJ0wkDDPZ5RB4eJ3EZtZd2LvIvA8tZIPzqthGyPgzTO3VKl8l5/pw27b+77/fj8y/HcZhWn5f3N5Ui1rTtZeeorcaNg/JVjJu3LMzPGUhiuXSO6pxCKsxFRSTpf/f0Q49NCvR7QosW+ZAcjQlTi6XTjOGNrGD+C6wwZs1jjyw8xxDNLRmOuydho4uCpCJZVIBhwGzWkrukxdNnW722Wli9uEBpniCJ6QfY8Ov2aur91poIJDsdowNlAbVTJquW3RJzGMJRAe4mtFMzbgHqtTOQ/2HVnhVZwedgUJbCh8+DGg0B95XPWhZ90jbHqE0PIR5Par1JDsY23GWOoCxw8m4UGZEL3gOG3+yE2omB/K0APUFZW7Y5Nt65ylQVW5AHDKblPy1NJzSSo+61J+6jhxrBUSW21LBmAlnzgfC5xDs3Iobf28Z9kWzhEMXdMI9/dqfnedUsHpOzGVK+3katmNFlQhvQgh2HQ+/a3KNtBt6BgvzRTLACKxiHYyXOT8espINSl2UWL06QXsFNKKF5dTEyvEmzbofcgjR22tjcWKVCrPSKYG0YHG3AjbIcnn+U3efcQkeyuCbVJjjWP2zWj9pK4T2PuMUKrWlMF/6ItaPDDKLGGoJOOigtCC70mlDkXaF0km19RL5tIgTMXzNTZJAQ3F+xsMab8QHcTooqmJ5EPztwLiv/uC7j9RUU8pbukn1osGx8Bf5XBXAIP3OXTRaSg/Q56PEU2GBeXetegGcWceG7KBYSrS9UE6r+g3ZPl6dEdVwdNLXmRtITLHZBCumQjt2IW1o3zDLzQt2CKdh5U0eJsoz9KvG0BWGuWsPeFcuUHxFZBR23lLo8PZpV5/t+99ML002w7a80ZPFMZgnPsicy1nIYHBautLQsCSdUm7AAtCYf0zL9L72Kl+JK2aVryO77BJ9CPgsJUhmRQppjulvqDVt9rl6+M/6aqNWTFN43qW0XdP9cRoz6QxxbJOPRFDwgJPYrETlgGakB47CbVW5+Yst3x+hvGQI1gd84T7ZNaJzyzn9Srv9adyPFgVW6GNsnlcs0RRTY6WN5njNcxtL1AtaJgHgb54GtVFAKRQDZB7MUIoPGUpTHihw4tRphYGBGyLSa4HxZ7S76BLBReDj2D77sdO0QhyQIsCS8Zngizotf7rUXUEEzIQU9KrjEuStRuFbWpW6bED7vbODnR9uJR/FkqNHdaBxvALkMKRCQ/oq/UTx5FMDd2GCBT2oS2cehBAoaC9qkAfX2xsZATzXoAf4C+CW1yoyFmcr742oE4xFk3BcqmIcehy8i2ev8IEIWQ9ehixzqdbHKfUGLgCgr3PTiNfc+RECyJU2idnyAnog/3Yqd2zLCliPWYcXrzex2TVct/ZN86shQWP/8KUPa0OCkWhK+Q9vh3s2OTZIG/7LNQYrrg56C6dD+kcTci1g/qffVOo403+f6QoFdYCMNWVLB/O5e5tnUSNEDfP4sPKUgWQhxB53HcwggolBgkqhkiG9w0BBwGgggoWBIIKEjCCCg4wggoKBgsqhkiG9w0BDAoBAqCCCbEwggmtMFcGCSqGSIb3DQEFDTBKMCkGCSqGSIb3DQEFDDAcBAgBS68zHNqTgQICCAAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEEIzB1wJPWoUGAgMgm6n2/YwEgglQGaOJRIkIg2BXvJJ0n+689/+9iDt8J3S48R8cA7E1hKMSlsXBzFK6VinIcjESDNf+nkiRpBIN1rmuP7WY81S7GWegXC9dp/ya4e8Y8HVqpdf+yhPhkaCn3CpYGcH3c+To3ylmZ5cLpD4kq1ehMjHr/D5SVxaq9y3ev016bZaVICzZ0+9PG8+hh2Fv/HK4dqsgjX1bPAc2kqnYgoCaF/ETtcSoiCLavMDFTFCdVeVQ/7TSSuFlT/HJRXscfdmjkYDXdKAlwejCeb4F4T2SfsiO5VVf15J/tgGsaZl77UiGWYUAXJJ/8TFTxVXYOTIOnBOhFBSH+uFXgGuh+S5eq2zq/JZVEs2gWgTz2Yn0nMpuHzLfiOKLRRk4pIgpZ3Lz44VBzSXjE2KaAopgURfoRQz25npPW7Ej/xjetFniAkxx2Ul/KTNu9Nu8SDR7zdbdJPK5hKh9Ix66opKg7yee2aAXDivedcKRaMpNApHMbyUYOmZgxc+qvcf+Oe8AbV6X8vdwzvBLSLAovuP+OubZ4G7Dt08dVAERzFOtxsjWndxYgiSbgE0onX37pJXtNasBSeOfGm5RIbqsxS8yj/nZFw/iyaS7CkTbQa8zAutGF7Q++0u0yRZntI9eBgfHoNLSv9Be9uD5PlPetBC7n3PB7/3zEiRQsuMH8TlcKIcvOBB56Alpp8kn4sAOObmdSupIjKzeW3/uj8OpSoEyJ+MVjbwCmAeq5sUQJwxxa6PoI9WHzeObI9PGXYNsZd1O7tAmnL00yJEQP5ZGMexGiQviL6qk7RW6tUAgZQP6L9cPetJUUOISwZNmLuoitPmlomHPNmjADDh+rFVxeNTviZY0usOxhSpXuxXCSlgRY/197FSms0RmDAjw/AEnwSCzDRJp/25n6maEJ8rWxQPZwcCfObsMfEtxyLkN4Qd62TDlTgekyxnRepeZyk8rXnwDDzK6GZRmXefBNq7dHFqp7eHG25EZJVotE43x3AKf/cHrf0QmmzkNROWadUitWPAxHjEZax9oVST5+pPJeJbROW6ItoBVWTSKLndxzn8Kyg/J6itaRUU4ZQ3QHPanO9uqqvjJ78km6PedoMyrk+HNkWVOeYD0iUV3caeoY+0/S+wbvMidQC0x6Q7BBaHYXCoH7zghbB4hZYyd7zRJ9MCW916QID0Bh+DX7sVBua7rLAMJZVyWfIvWrkcZezuPaRLxZHK54+uGc7m4R95Yg9V/Juk0zkHBUY66eMAGFjXfBl7jwg2ZQWX+/kuALXcrdcSWbQ6NY7en60ujm49A8h9CdO6gFpdopPafvocGgCe5D29yCYGAPp9kT+ComEXeHeLZ0wWlP77aByBdO9hJjXg7MSqWN8FuICxPsKThXHzH68Zi+xqqAzyt5NaVnvLvtMAaS4BTifSUPuhC1dBmTkv0lO36a1LzKlPi4kQnYI6WqOKg5bqqFMnkc+/y5UMlGO7yYockQYtZivVUy6njy+Gum30T81mVwDY21l7KR2wCS7ItiUjaM9X+pFvEa/MznEnKe0O7di8eTnxTCUJWKFAZO5n/k7PbhQm9ZGSNXUxeSwyuVMRj4AwW3OJvHXon8dlt4TX66esCjEzZKtbAvWQY68f2xhWZaOYbxDmpUGvG7vOPb/XZ8XtE57nkcCVNxtLKk47mWEeMIKF+0AzfMZB+XNLZFOqr/svEboPH98ytQ5j1sMs54rI9MHKWwSPrh/Wld18flZPtnZZHjLg5AAM0PX7YZyp3tDqxfLn/Uw+xOV/4RPxY3qGzvQb1CdNXUBSO9J8imIfSCySYsnpzdi3MXnAaA59YFi5WVLSTnodtyEdTeutO9UEw6q+ddjjkBzCPUOArc/60jfNsOThjeQvJWvzmm6BmrLjQmrQC3p8eD6kT56bDV6l2xkwuPScMfXjuwPLUZIK8THhQdXowj2CAi7qAjvHJfSP5pA4UU/88bI9SW07YCDmqTzRhsoct4c+NluqSHrgwRDcOsXGhldMDxF4mUGfObMl+gva2Sg+aXtnQnu90Z9HRKUNIGSJB7UBOKX/0ziQdB3F1KPmer4GQZrAq/YsVClKnyw3dkslmNRGsIcQET3RB0UEI5g4p0bcgL9kCUzwZFZ6QW2cMnl7oNlMmtoC+QfMo+DDjsbjqpeaohoLpactsDvuqXYDef62the/uIEEu6ezuutcwk5ABvzevAaJGSYCY090jeB865RDQUf7j/BJANYOoMtUwn/wyPK2vcMl1AG0fwYrL1M4brnVeMBcEpsbWfhzWgMObZjojP52hQBjl0F+F3YRfk0k1Us4hGYkjQvdMR3YJBnSll5A9dN5EhL53f3eubBFdtwJuFdkfNOsRNKpL0TcA//6HsJByn5K+KlOqkWkhooIp4RB6UBHOmSroXoeiMdopMm8B7AtiX7aljLD0ap480GAEZdvcR55UGpHuy8WxYmWZ3+WNgHNa4UE4l3W1Kt7wrHMVd0W6byxhKHLiGO/8xI1kv2gCogT+E7bFD20E/oyI9iaWQpZXOdGTVl2CqkCFGig+aIFcDADqG/JSiUDg/S5WucyPTqnFcmZGE+jhmfI78CcsB4PGT1rY7CxnzViP38Rl/NCcT9dNfqhQx5Ng5JlBsV3Ets0Zy6ZxIAUG5BbMeRp3s8SmbHoFvZMBINgoETdaw6AhcgQddqh/+BpsU7vObu6aehSyk9xGSeFgWxqOV8crFQpbl8McY7ONmuLfLjPpAHjv8s5TsEZOO+mu1LeSgYXuEGN0fxklazKGPRQe7i4Nez1epkgR6+/c7Ccl9QOGHKRpnZ4Mdn4nBCUzXn9jH80vnohHxwRLPMfMcArWKxY3TfRbazwQpgxVV9qZdTDXqRbnthtdrfwDBj2/UcPPjt87x8/qSaEWT/u9Yb65Gsigf0x7W7beYo0sWpyJJMJQL/U0cGM+kaFU6+fiPHz8jO1tkdVFWb+zv6AlzUuK6Q6EZ7F+DwqLTNUK1zDvpPMYKwt1b4bMbIG7liVyS4CQGpSNwY58QQ0TThnS1ykEoOlC74gB7Rcxp/pO8Ov2jHz1fY7CF7DmZeWqeRNATUWZSayCYzArTUZeNK4EPzo2RAfMy/5kP9RA11FoOiFhj5Ntis8kn2YRx90vIOH9jhJiv6TcqceNR+nji0Flzdnule6myaEXIoXKqp5RVVgJTqwQzWc13+0xRjAfBgkqhkiG9w0BCRQxEh4QAHQAZQBzAHQALgBjAG8AbTAjBgkqhkiG9w0BCRUxFgQUwpGMjmJDPDoZdapGelDCIEATkm0wQTAxMA0GCWCGSAFlAwQCAQUABCDRnldCcEWY+iPEzeXOqYhJyLUH7Geh6nw2S5eZA1qoTgQI4ezCrgN0h8cCAggA",
			"-var=certificate_test_password=Password01!",
			"-var=account_google=secretgoeshere",
			"-var=account_azure=secretgoeshere",
			"-var=account_aws_account=secretgoeshere",
			"-var=account_usernamepasswordaccount=secretgoeshere",
			"-var=account_ssh_cert=whatever",
			"-var=account_ssh=LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJGd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFRRUF5c25PVXhjN0tJK2pIRUc5RVEwQXFCMllGRWE5ZnpZakZOY1pqY1dwcjJQRkRza25oOUpTCm1NVjVuZ2VrbTRyNHJVQU5tU2dQMW1ZTGo5TFR0NUVZa0N3OUdyQ0paNitlQTkzTEowbEZUamFkWEJuQnNmbmZGTlFWYkcKZ2p3U1o4SWdWQ2oySXE0S1hGZm0vbG1ycEZQK2Jqa2V4dUxwcEh5dko2ZmxZVjZFMG13YVlneVNHTWdLYy9ubXJaMTY0WApKMStJL1M5NkwzRWdOT0hNZmo4QjM5eEhZQ0ZUTzZEQ0pLQ3B0ZUdRa0gwTURHam84d3VoUlF6c0IzVExsdXN6ZG0xNmRZCk16WXZBSWR3emZ3bzh1ajFBSFFOendDYkIwRmR6bnFNOEpLV2ZrQzdFeVVrZUl4UXZmLzJGd1ZyS0xEZC95ak5PUmNoa3EKb2owNncySXFad0FBQThpS0tqT3dpaW96c0FBQUFBZHpjMmd0Y25OaEFBQUJBUURLeWM1VEZ6c29qNk1jUWIwUkRRQ29IWgpnVVJyMS9OaU1VMXhtTnhhbXZZOFVPeVNlSDBsS1l4WG1lQjZTYml2aXRRQTJaS0EvV1pndVAwdE8za1JpUUxEMGFzSWxuCnI1NEQzY3NuU1VWT05wMWNHY0d4K2Q4VTFCVnNhQ1BCSm53aUJVS1BZaXJncGNWK2IrV2F1a1UvNXVPUjdHNHVta2ZLOG4KcCtWaFhvVFNiQnBpREpJWXlBcHorZWF0blhyaGNuWDRqOUwzb3ZjU0EwNGN4K1B3SGYzRWRnSVZNN29NSWtvS20xNFpDUQpmUXdNYU9qekM2RkZET3dIZE11VzZ6TjJiWHAxZ3pOaThBaDNETi9Dank2UFVBZEEzUEFKc0hRVjNPZW96d2twWitRTHNUCkpTUjRqRkM5Ly9ZWEJXc29zTjMvS00wNUZ5R1NxaVBUckRZaXBuQUFBQUF3RUFBUUFBQVFFQXdRZzRqbitlb0kyYUJsdk4KVFYzRE1rUjViMU9uTG1DcUpEeGM1c2N4THZNWnNXbHBaN0NkVHk4ckJYTGhEZTdMcUo5QVVub0FHV1lwdTA1RW1vaFRpVwptVEFNVHJCdmYwd2xsdCtJZVdvVXo3bmFBbThQT1psb29MbXBYRzh5VmZKRU05aUo4NWtYNDY4SkF6VDRYZ1JXUFRYQ1JpCi9abCtuWUVUZVE4WTYzWlJhTVE3SUNmK2FRRWxRenBYb21idkxYM1RaNmNzTHh5Z3Eza01aSXNJU0lUcEk3Y0tsQVJ0Rm4KcWxKRitCL2JlUEJkZ3hIRVpqZDhDV0NIR1ZRUDh3Z3B0d0Rrak9NTzh2b2N4YVpOT0hZZnBwSlBCTkVjMEVKbmduN1BXSgorMVZSTWZKUW5SemVubmE3VHdSUSsrclZmdkVaRmhqamdSUk85RitrMUZvSWdRQUFBSUVBbFFybXRiV2V0d3RlWlZLLys4CklCUDZkcy9MSWtPb3pXRS9Wckx6cElBeHEvV1lFTW1QK24wK1dXdWRHNWpPaTFlZEJSYVFnU0owdTRxcE5JMXFGYTRISFYKY2oxL3pzenZ4RUtSRElhQkJGaU81Y3QvRVQvUTdwanozTnJaZVdtK0dlUUJKQ0diTEhSTlQ0M1ZpWVlLVG82ZGlGVTJteApHWENlLzFRY2NqNjVZQUFBQ0JBUHZodmgzb2Q1MmY4SFVWWGoxeDNlL1ZFenJPeVloTi9UQzNMbWhHYnRtdHZ0L0J2SUhxCndxWFpTT0lWWkZiRnVKSCtORHNWZFFIN29yUW1VcGJxRllDd0IxNUZNRGw0NVhLRm0xYjFyS1c1emVQK3d0M1hyM1p0cWsKRkdlaUlRMklSZklBQjZneElvNTZGemdMUmx6QnB0bzhkTlhjMXhtWVgyU2Rhb3ZwSkRBQUFBZ1FET0dwVE9oOEFRMFoxUwpzUm9vVS9YRTRkYWtrSU5vMDdHNGI3M01maG9xbkV1T01LM0ZRVStRRWUwYWpvdWs5UU1QNWJzZU1CYnJNZVNNUjBRWVBCClQ4Z0Z2S2VISWN6ZUtJTjNPRkRaRUF4TEZNMG9LbjR2bmdHTUFtTXUva2QwNm1PZnJUNDRmUUh1ajdGNWx1QVJHejRwYUwKLzRCTUVkMnFTRnFBYzZ6L0RRQUFBQTF0WVhSMGFFQk5ZWFIwYUdWM0FRSURCQT09Ci0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=",
			"-var=account_subscription_cert=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",
			"-var=account_token=whatever",
			"-var=account_ec2_sydney=LS0tLS1CRUdJTiBFTkNSWVBURUQgUFJJVkFURSBLRVktLS0tLQpNSUlKbkRCT0Jna3Foa2lHOXcwQkJRMHdRVEFwQmdrcWhraUc5dzBCQlF3d0hBUUlwNEUxV1ZrejJEd0NBZ2dBCk1Bd0dDQ3FHU0liM0RRSUpCUUF3RkFZSUtvWklodmNOQXdjRUNIemFuVE1QbHA4ZkJJSUpTSncrdW5BL2ZaVFUKRGdrdWk2QnhOY0REUFg3UHZJZmNXU1dTc3V3YWRhYXdkVEdjY1JVd3pGNTNmRWJTUXJBYzJuWFkwUWVVcU1wcAo4QmdXUUthWlB3MEdqck5OQVJaTy9QYklxaU5ERFMybVRSekZidzREcFY5aDdlblZjL1ZPNlhJdzlxYVYzendlCnhEejdZSkJ2ckhmWHNmTmx1blErYTZGdlRUVkVyWkE1Ukp1dEZUVnhUWVR1Z3lvWWNXZzAzQWlsMDh3eDhyTHkKUkgvTjNjRlEzaEtLcVZuSHQvdnNZUUhTMnJJYkt0RTluelFPWDRxRDdVYXM3Z0c0L2ZkcmZQZjZFWTR1aGpBcApUeGZRTDUzcTBQZG85T09MZlRReFRxakVNaFpidjV1aEN5d0N2VHdWTmVLZ2MzN1pqdDNPSjI3NTB3U2t1TFZvCnllR0VaQmtML1VONjJjTUJuYlFsSTEzR2FXejBHZ0NJNGkwS3UvRmE4aHJZQTQwcHVFdkEwZFBYcVFGMDhYbFYKM1RJUEhGRWdBWlJpTmpJWmFyQW00THdnL1F4Z203OUR1SVM3VHh6RCtpN1pNSmsydjI1ck14Ly9MMXNNUFBtOQpWaXBwVnpLZmpqRmpwTDVjcVJucC9UdUZSVWpHaDZWMFBXVVk1eTVzYjJBWHpuSGZVd1lqeFNoUjBKWXpXejAwCjNHbklwNnlJa1UvL3dFVGJLcVliMjd0RjdETm1WMUxXQzl0ell1dm4yK2EwQkpnU0Jlc3c4WFJ1WWorQS92bVcKWk1YbkF2anZXR3RBUzA4d0ZOV3F3QUtMbzJYUHBXWGVMa3BZUHo1ZnY2QnJaNVNwYTg4UFhsa1VmOVF0VHRobwprZFlGOWVMdk5hTXpSSWJhbmRGWjdLcHUvN2I3L0tDWE9rMUhMOUxvdEpwY2tJdTAxWS81TnQwOHp5cEVQQ1RzClVGWG5DODNqK2tWMktndG5XcXlEL2k3Z1dwaHJSK0IrNE9tM3VZU1RuY042a2d6ZkV3WldpUVA3ZkpiNlYwTHoKc29yU09sK2g2WDRsMC9oRVdScktVQTBrOXpPZU9TQXhlbmpVUXFReWdUd0RqQTJWbTdSZXI2ZElDMVBwNmVETgpBVEJ0ME1NZjJJTytxbTJtK0VLd1FVSXY4ZXdpdEpab016MFBaOHB6WEM0ZFMyRTErZzZmbnE2UGJ5WWRISDJnCmVraXk4Y2duVVJmdHJFaVoyMUxpMWdpdTJaeVM5QUc0Z1ZuT0E1Y05oSzZtRDJUaGl5UUl2M09yUDA0aDFTNlEKQUdGeGJONEhZK0tCYnVITTYwRG1PQXR5c3o4QkJheHFwWjlXQkVhV01ubFB6eEI2SnFqTGJrZ1BkQ2wycytUWAphcWx0UDd6QkpaenVTeVNQc2tQR1NBREUvaEF4eDJFM1RQeWNhQlhQRVFUM2VkZmNsM09nYXRmeHBSYXJLV09PCnFHM2lteW42ZzJiNjhWTlBDSnBTYTNKZ1Axb0NNVlBpa2RCSEdSVUV3N2dXTlJVOFpXRVJuS292M2c0MnQ4dkEKU2Z0a3VMdkhoUnlPQW91SUVsNjJIems0WC9CeVVOQ2J3MW50RzFQeHpSaERaV2dPaVhPNi94WFByRlpKa3BtcQpZUUE5dW83OVdKZy9zSWxucFJCdFlUbUh4eU9mNk12R2svdXlkZExkcmZ6MHB6QUVmWm11YTVocWh5M2Y4YlNJCmpxMlJwUHE3eHJ1Y2djbFAwTWFjdHkrbm9wa0N4M0lNRUE4NE9MQ3dxZjVtemtwY0U1M3hGaU1hcXZTK0dHZmkKZlZnUGpXTXRzMFhjdEtCV2tUbVFFN3MxSE5EV0g1dlpJaDY2WTZncXR0cjU2VGdtcHRLWHBVdUJ1MEdERFBQbwp1aGI4TnVRRjZwNHNoM1dDbXlzTU9uSW5jaXRxZWE4NTFEMmloK2lIY3VqcnJidkVYZGtjMnlxUHBtK3Q3SXBvCm1zWkxVemdXRlZpNWY3KzZiZU56dGJ3T2tmYmdlQVAyaklHTzdtR1pKWWM0L1d1eXBqeVRKNlBQVC9IMUc3K3QKUTh5R3FDV3BzNFdQM2srR3hrbW90cnFROFcxa0J1RDJxTEdmSTdMMGZUVE9lWk0vQUZ1VDJVSkcxKzQ2czJVVwp2RlF2VUJmZ0dTWlh3c1VUeGJRTlZNaTJib1BCRkNxbUY2VmJTcmw2YVgrSm1NNVhySUlqUUhGUFZWVGxzeUtpClVDUC9PQTJOWlREdW9IcC9EM0s1Qjh5MlIyUTlqZlJ0RkcwL0dnMktCbCtObzdTbXlPcWlsUlNkZ1VJb0p5QkcKRGovZXJ4ZkZNMlc3WTVsNGZ2ZlNpdU1OZmlUTVdkY3cxSStnVkpGMC9mTHRpYkNoUlg0OTlIRWlXUHZkTGFKMwppcDJEYU9ReS9QZG5zK3hvaWlMNWtHV25BVUVwanNjWno0YU5DZFowOXRUb1FhK2RZd3g1R1ovNUtmbnVpTURnClBrWjNXalFpOVlZRWFXbVIvQ2JmMjAyRXdoNjdIZzVqWE5kb0RNendXT0V4RFNkVFFYZVdzUUI0LzNzcjE2S2MKeitGN2xhOXhHVEVhTDllQitwcjY5L2JjekJLMGVkNXUxYUgxcXR3cjcrMmliNmZDdlMyblRGQTM1ZG50YXZlUwp4VUJVZ0NzRzVhTTl4b2pIQ0o4RzRFMm9iRUEwUDg2SFlqZEJJSXF5U0txZWtQYmFybW4xR1JrdUVlbU5hTVdyCkM2bWZqUXR5V2ZMWnlSbUlhL1dkSVgzYXhqZHhYa3kydm4yNVV6MXZRNklrNnRJcktPYUJnRUY1cmYwY014dTUKN1BYeTk0dnc1QjE0Vlcra2JqQnkyY3hIajJhWnJEaE53UnVQNlpIckg5MHZuN2NmYjYwU0twRWxxdmZwdlN0VQpvQnVXQlFEUUE3bHpZajhhT3BHend3LzlYTjI5MGJrUnd4elVZRTBxOVl4bS9VSHJTNUlyRWtKSml2SUlEb3hICjF4VTVLd2ErbERvWDJNcERrZlBQVE9XSjVqZG8wbXNsN0dBTmc1WGhERnBpb2hFMEdSS2lGVytYcjBsYkJKU2oKUkxibytrbzhncXU2WHB0OWU4U0Y5OEJ4bFpEcFBVMG5PcGRrTmxwTVpKYVlpaUUzRjRFRG9DcE56bmxpY2JrcApjZ2FrcGVrbS9YS21RSlJxWElXci8wM29SdUVFTXBxZzlRbjdWRG8zR0FiUTlnNUR5U1Bid0xvT25xQ0V3WGFJCkF6alFzWU4rc3VRd2FqZHFUcEthZ1FCbWRaMmdNZDBTMTV1Ukt6c2wxOHgzK1JabmRiNWoxNjNuV0NkMlQ5VDgKald3NURISDgvVUFkSGZoOHh0RTJ6bWRHbEg5T3I5U2hIMzViMWgxVm8rU2pNMzRPeWpwVjB3TmNVL1psOTBUdAp1WnJwYnBwTXZCZUVmRzZTczVXVGhySm9LaGl0RkNwWlVqaDZvdnk3Mzd6ditKaUc4aDRBNG1GTmRPSUtBd0I0Cmp2Nms3V3poUVlEa2Q0ZXRoajNndVJCTGZQNThNVEJKaWhZemVINkUzclhjSGE5b0xnREgzczd4bU8yVEtUY24Kd3VIM3AvdC9WWFN3UGJ0QXBXUXdTRFNKSnA5WkF4S0Q1eVdmd3lTU2ZQVGtwM2c1b2NmKzBhSk1Kc2FkU3lwNQpNR1Vic1oxd1hTN2RXMDhOYXZ2WmpmbElNUm8wUFZDbkRVcFp1bjJuekhTRGJDSjB1M0ZYd1lFQzFFejlJUnN0ClJFbDdpdTZQRlVMSldSU0V0SzBKY1lLS0ltNXhQWHIvbTdPc2duMUNJL0F0cTkrWEFjODk1MGVxeTRwTFVQYkYKZkhFOFhVYWFzUU82MDJTeGpnOTZZaWJ3ZnFyTDF2Vjd1MitUYzJleUZ1N3oxUGRPZDQyWko5M2wvM3lOUW92egora0JuQVdObzZ3WnNKSitHNDZDODNYRVBLM0h1bGw1dFg2UDU4NUQ1b3o5U1oyZGlTd1FyVFN1THVSL0JCQUpVCmd1K2FITkJGRmVtUXNEL2QxMllud1h3d3FkZXVaMDVmQlFiWUREdldOM3daUjJJeHZpd1E0bjZjZWl3OUZ4QmcKbWlzMFBGY2NZOWl0SnJrYXlWQVVZUFZ3Sm5XSmZEK2pQNjJ3UWZJWmhhbFQrZDJpUzVQaDEwdWlMNHEvY1JuYgo1c1Mvc2o0Tm5QYmpxc1ZmZWlKTEh3PT0KLS0tLS1FTkQgRU5DUllQVEVEIFBSSVZBVEUgS0VZLS0tLS0K",
			"-var=account_ec2_sydney_cert=whatever",
			"-var=account_subscription_cert=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",
			"-var=target_service_fabric=whatever",
			"-var=account_sales_account=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Certificate]{}
			err := octopusClient.GetAllResources("Certificates", &collection)

			if err != nil {
				return err
			}

			// Certificates

			certificate := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "Test"
			})

			if len(certificate) != 1 {
				return errors.New("Space must have an tenant called \"Test\" in space " + recreatedSpaceId)
			}

			if certificate[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The certificate must be have a tenant participation of \"Tenanted\" (was \"" + certificate[0].TenantedDeploymentParticipation + "\")")
			}

			if len(certificate[0].TenantTags) != 1 {
				return errors.New("The certificate must have one tenant tags")
			}

			err = octopusClient.GetAllResources("Accounts", &collection)

			if err != nil {
				return err
			}

			// AWS

			awsAccount := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "AWS Account"
			})

			if len(awsAccount) != 1 {
				return errors.New("Space must have an account called \"AWS Account\" in space " + recreatedSpaceId)
			}

			if awsAccount[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + awsAccount[0].TenantedDeploymentParticipation + "\")")
			}

			if len(awsAccount[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// Azure

			azureAccount := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "Azure"
			})

			if len(azureAccount) != 1 {
				return errors.New("Space must have an account called \"Azure\" in space " + recreatedSpaceId)
			}

			if azureAccount[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + azureAccount[0].TenantedDeploymentParticipation + "\")")
			}

			if len(azureAccount[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// Google

			google := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "Google"
			})

			if len(google) != 1 {
				return errors.New("Space must have an account called \"Google\" in space " + recreatedSpaceId)
			}

			if google[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + google[0].TenantedDeploymentParticipation + "\")")
			}

			if len(google[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// SSH

			ssh := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "SSH"
			})

			if len(google) != 1 {
				return errors.New("Space must have an account called \"SSH\" in space " + recreatedSpaceId)
			}

			if ssh[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + ssh[0].TenantedDeploymentParticipation + "\")")
			}

			if len(ssh[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// Token

			token := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "Token"
			})

			if len(token) != 1 {
				return errors.New("Space must have an account called \"Token\" in space " + recreatedSpaceId)
			}

			if token[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + token[0].TenantedDeploymentParticipation + "\")")
			}

			if len(token[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// Username and Password

			userPass := lo.Filter(collection.Items, func(item octopus.Certificate, index int) bool {
				return item.Name == "UsernamePasswordAccount"
			})

			if len(userPass) != 1 {
				return errors.New("Space must have an account called \"UsernamePasswordAccount\" in space " + recreatedSpaceId)
			}

			if userPass[0].TenantedDeploymentParticipation != "Tenanted" {
				return errors.New("The account must be have a tenant participation of \"Tenanted\" (was \"" + userPass[0].TenantedDeploymentParticipation + "\")")
			}

			if len(userPass[0].TenantTags) != 1 {
				return errors.New("The account must have one tenant tags")
			}

			// Project

			projectCollection := octopus.GeneralCollection[octopus.Project]{}
			err = octopusClient.GetAllResources("Projects", &projectCollection)

			if err != nil {
				return err
			}

			project := lo.Filter(projectCollection.Items, func(item octopus.Project, index int) bool {
				return item.Name == "Test"
			})

			if len(project) != 1 {
				return errors.New("Space must have a project called \"Test\" in space " + recreatedSpaceId)
			}

			deploymentProcess := octopus.DeploymentProcess{}
			_, err = octopusClient.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(project[0].DeploymentProcessId), &deploymentProcess)

			if err != nil {
				return err
			}

			if len(deploymentProcess.Steps) != 1 {
				return errors.New("Space must have a project called \"Test\" with a single step in space " + recreatedSpaceId)
			}

			if len(deploymentProcess.Steps[0].Actions) != 1 {
				return errors.New("Space must have a project called \"Test\" with a single step with a single action in space " + recreatedSpaceId)
			}

			if len(deploymentProcess.Steps[0].Actions[0].TenantTags) != 1 {
				return errors.New("Deployment process must have an action with 1 tenant tag")
			}

			return nil
		})
}

// TestSingleRunbookExport verifies that a single runbook can be reimported with the correct settings.
func TestSingleRunbookExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/69-runbookexport/space_creation",
		"../test/terraform/69-runbookexport/space_prepopulation",
		"../test/terraform/69-runbookexport/space_population",
		"../test/terraform/69-runbookexport/space_creation",
		"../test/terraform/69-runbookexport/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			RunbookName: "Runbook",
			ProjectName: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			// Verify that the single project was exported
			err := func() error {
				runbookCollection := octopus.GeneralCollection[octopus.Runbook]{}
				err := octopusClient.GetAllResources("Runbooks", &runbookCollection)

				if err != nil {
					return err
				}

				if len(runbookCollection.Items) != 1 {
					return errors.New("There must only be one runbook")
				}

				if runbookCollection.Items[0].Name != "Runbook" {
					return errors.New("The runbook must be called \"Runbook\"")
				}

				return nil
			}()

			if err != nil {
				return err
			}

			return nil
		})
}

// TestLotsOfProjectExport verifies that a large number of projects (more than the API batch size) are processed
// correctly.
func TestLotsOfProjectExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/70-lotsofproject/space_creation",
		"../test/terraform/70-lotsofproject/space_population",
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			// Verify one of the last projects is exported correctly
			resourceName := "Test 99"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					if strutil.EmptyIfNil(v.Description) != "Test project 99" {
						return errors.New("The project must be have a description of \"Test project 99\" (was \"" + strutil.EmptyIfNil(v.Description) + "\")")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectWorkerPoolVariableExport verifies that a project with steps using a worker pool variables
// can be reimported with the correct settings.
func TestProjectWorkerPoolVariableExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/71-workerpoolvariable/space_creation",
		"../test/terraform/71-workerpoolvariable/space_population",
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{
			// The step in this example uses a worker pool variable, so the empty worker pool
			// reference must be ignored
			LookUpDefaultWorkerPools: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					variables := octopus.VariableSet{}
					_, err = octopusClient.GetSpaceResourceById("Variables", strutil.EmptyIfNil(v.VariableSetId), &variables)

					if err != nil {
						return err
					}

					if len(variables.Variables) != 1 {
						return errors.New("The project must have a single variable.")
					}

					if variables.Variables[0].Name != "WorkerPool" {
						return errors.New("The project must have a single variable called \"WorkerPool\".")
					}

					workerPoolResource := octopus.WorkerPool{}
					_, err = octopusClient.GetSpaceResourceById("WorkerPools", strutil.EmptyIfNil(variables.Variables[0].Value), &workerPoolResource)

					if err != nil {
						return err
					}

					if workerPoolResource.Name != "Static pool" {
						return errors.New("The project must have a variable referencing a worker pool called \"Static pool\".")
					}

					deploymentProcess := octopus.DeploymentProcess{}
					_, err := octopusClient.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(v.DeploymentProcessId), &deploymentProcess)

					if err != nil {
						return err
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].WorkerPoolVariable) != "#{WorkerPool}" {
						return errors.New("Deployment process should have a worker pool variable of \"#{WorkerPool}\".")
					}

					if deploymentProcess.Steps[0].Actions[0].WorkerPoolId != "" {
						return errors.New("Deployment process should have an empty worker pool.")
					}

					runbookCollection := octopus.GeneralCollection[octopus.Runbook]{}
					err = octopusClient.GetAllResources("Runbooks", &runbookCollection)

					if err != nil {
						return err
					}

					runbook := lo.Filter(runbookCollection.Items, func(item octopus.Runbook, index int) bool {
						return item.Name == "Runbook"
					})

					if len(runbook) != 1 {
						return errors.New("Should have created a runbook called \"Runbook\".")
					}

					runbookProcess := octopus.RunbookProcess{}
					_, err = octopusClient.GetSpaceResourceById("RunbookProcesses", strutil.EmptyIfNil(runbook[0].RunbookProcessId), &runbookProcess)

					if err != nil {
						return err
					}

					if strutil.EmptyIfNil(runbookProcess.Steps[0].Actions[0].WorkerPoolVariable) != "#{WorkerPool}" {
						return errors.New("Runbook step should have had a worker pool variable of \"#{WorkerPool}\".")
					}

					if runbookProcess.Steps[0].Actions[0].WorkerPoolId != "" {
						return errors.New("Runbook step should have had an empty worker pool ID.")
					}
				}
			}

			if !found {
				return errors.New("Space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectScheduledTriggerExport verifies that a project can be reimported with scheduled triggers
func TestProjectScheduledTriggerExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/72-projecttrigger/space_creation",
		"../test/terraform/72-projecttrigger/space_population",
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			environments := octopus.GeneralCollection[octopus.Environment]{}
			err = octopusClient.GetAllResources("Environments", &environments)

			if err != nil {
				return err
			}

			testEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Test"
			})

			developmentEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Development"
			})

			resourceName := "Test"

			project := lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
				return item.Name == resourceName
			})

			if len(project) != 1 {
				return errors.New("space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
			err = octopusClient.GetAllResources("Projects/"+project[0].Id+"/Triggers", &triggers)

			if err != nil {
				return err
			}

			onceDailyExample := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Once Daily example"
			})

			if len(onceDailyExample) != 1 {
				return errors.New("space must have an trigger called \"Once Daily example\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(onceDailyExample[0].Action.EnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Once Daily example\" must have an environment of Test")
			}

			if len(onceDailyExample[0].Action.SourceEnvironmentIds) != 0 {
				return errors.New("the trigger \"Once Daily example\" must have no source environments")
			}

			if onceDailyExample[0].Action.DestinationEnvironmentId != nil {
				return errors.New("the trigger \"Once Daily example\" must have no destination environment")
			}

			if strutil.EmptyIfNil(onceDailyExample[0].Filter.StartTime) != "2024-03-22T09:00:00.000Z" {
				return errors.New("the trigger \"Once Daily example\" must have a start time of \"2024-03-22T09:00:00.000Z\"")
			}

			if slices.Index(onceDailyExample[0].Filter.DaysOfWeek, "Monday") == -1 {
				return errors.New("the trigger \"Once Daily example\" must have a day of the week as \"Monday\"")
			}

			if slices.Index(onceDailyExample[0].Filter.DaysOfWeek, "Tuesday") == -1 {
				return errors.New("the trigger \"Once Daily example\" must have a day of the week as \"Tuesday\"")
			}

			if slices.Index(onceDailyExample[0].Filter.DaysOfWeek, "Wednesday") == -1 {
				return errors.New("the trigger \"Once Daily example\" must have a day of the week as \"Wednesday\"")
			}

			continuousExample := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Continuous"
			})

			if len(continuousExample) != 1 {
				return errors.New("Space must have an trigger called \"Continuous\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(continuousExample[0].Description) != "This is a continuous daily schedule" {
				return errors.New("the trigger \"Continuous\" must have a description \"This is a continuous daily schedule\"")
			}

			if strutil.EmptyIfNil(continuousExample[0].Action.EnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Continuous\" must have an environment of Test")
			}

			if strutil.EmptyIfNil(continuousExample[0].Filter.Interval) != "OnceHourly" {
				return errors.New("the trigger \"Continuous\" must have an interval of \"OnceHourly\"")
			}

			if intutil.ZeroIfNil(continuousExample[0].Filter.HourInterval) != 3 {
				return errors.New("the trigger \"Continuous\" must have an hourly interval of 3")
			}

			if slices.Index(continuousExample[0].Filter.DaysOfWeek, "Monday") == -1 {
				return errors.New("the trigger \"Continuous\" must have a day of the week as \"Monday\"")
			}

			if slices.Index(continuousExample[0].Filter.DaysOfWeek, "Tuesday") == -1 {
				return errors.New("the trigger \"Continuous\" must have a day of the week as \"Tuesday\"")
			}

			if slices.Index(continuousExample[0].Filter.DaysOfWeek, "Friday") == -1 {
				return errors.New("the trigger \"Continuous\" must have a day of the week as \"Friday\"")
			}

			deployLatest := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Deploy Latest"
			})

			if len(deployLatest) != 1 {
				return errors.New("space must have an trigger called \"Deploy Latest\" in space " + recreatedSpaceId)
			}

			if len(deployLatest[0].Action.SourceEnvironmentIds) != 1 {
				return errors.New("the trigger \"Deploy Latest\" must have 1 source environment")
			}

			if deployLatest[0].Action.SourceEnvironmentIds[0] != developmentEnvironment[0].Id {
				return errors.New("the trigger \"Deploy Latest\" must have 1 source environment of Development")
			}

			if strutil.EmptyIfNil(deployLatest[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Deploy Latest\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if strutil.EmptyIfNil(deployLatest[0].Action.DestinationEnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Deploy Latest\" must have an environment of Test")
			}

			if !boolutil.FalseIfNil(deployLatest[0].Action.ShouldRedeployWhenReleaseIsCurrent) {
				return errors.New("the trigger \"Deploy Latest\" must redeploy when release is current")
			}

			deployNew := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Deploy New"
			})

			if len(deployNew) != 1 {
				return errors.New("space must have an trigger called \"Deploy New\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(deployNew[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Deploy New\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if strutil.EmptyIfNil(deployNew[0].Action.EnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Deploy New\" must have an environment of Test")
			}

			runbook := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Runbook"
			})

			if len(runbook) != 1 {
				return errors.New("space must have an trigger called \"Runbook\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(runbook[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Runbook\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, testEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Test")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, developmentEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Development")
			}

			dateOfMonth := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Date Of Month"
			})

			if len(dateOfMonth) != 1 {
				return errors.New("space must have an trigger called \"Date Of Month\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(dateOfMonth[0].Filter.MonthlyScheduleType) != "DateOfMonth" {
				return errors.New("the trigger \"Date Of Month\" must have a monlthy schedule type of \"DateOfMonth\"")
			}

			if strutil.EmptyIfNil(dateOfMonth[0].Filter.StartTime) != "2024-03-22T09:00:00.000Z" {
				return errors.New("the trigger \"Date Of Month\" must have a start time of \"2024-03-22T09:00:00.000Z\"")
			}

			if strutil.EmptyIfNil(dateOfMonth[0].Filter.DateOfMonth) != "1" {
				return errors.New("the trigger \"Date Of Month\" must have a date of month of \"1\"")
			}

			dayOfMonth := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Day Of Month"
			})

			if len(dayOfMonth) != 1 {
				return errors.New("space must have an trigger called \"Day Of Month\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(dayOfMonth[0].Filter.MonthlyScheduleType) != "DayOfMonth" {
				return errors.New("the trigger \"Day Of Month\" must have a monthly schedule type of \"DateOfMonth\"")
			}

			if strutil.EmptyIfNil(dayOfMonth[0].Filter.StartTime) != "2024-03-22T09:00:00.000Z" {
				return errors.New("the trigger \"Day Of Month\" must have a start time of \"2024-03-22T09:00:00.000Z\"")
			}

			if strutil.EmptyIfNil(dayOfMonth[0].Filter.DayNumberOfMonth) != "1" {
				return errors.New("the trigger \"Day Of Month\" must have a day number of month of \"1\"")
			}

			if strutil.EmptyIfNil(dayOfMonth[0].Filter.DayOfWeek) != "Monday" {
				return errors.New("the trigger \"Day Of Month\" must have a day of week of \"Monday\"")
			}

			timeZone := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Time Zone Example"
			})

			if len(timeZone) != 1 {
				return errors.New("space must have an trigger called \"Time Zone Example\" in space " + recreatedSpaceId)
			}

			// At an API level, start and end times are stored in UTC. This is not entirely accurate,
			// as what this value represents is actually a time with no timezone. The timezone is
			// defined by the separate timezone field. I assume what happens here is the timezone
			// component of this UTC time is stripped and the timezone from the separate field
			// is appended.
			if strutil.EmptyIfNil(timeZone[0].Filter.RunAfter) != "2024-03-22T09:00:00.000Z" {
				return errors.New("the trigger \"Time Zone Example\" must have a start time of \"2024-03-22T09:00:00.000Z\"")
			}

			if strutil.EmptyIfNil(timeZone[0].Filter.Timezone) != "E. Australia Standard Time" {
				return errors.New("the trigger \"Time Zone Example\" must have a timezone of \"E. Australia Standard Time\"")
			}

			return nil
		})
}

// TestSingleProjectScheduledTriggerExport verifies that a single project can be reimported with scheduled triggers.
// We defer to the TestProjectScheduledTriggerExport test to verify that triggers are created with the correct values.
// This test is focused on ensuring environments are recursivly exported.
func TestSingleProjectScheduledTriggerExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/72-projecttrigger/space_creation",
		"../test/terraform/72-projecttrigger/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			environments := octopus.GeneralCollection[octopus.Environment]{}
			err = octopusClient.GetAllResources("Environments", &environments)

			if err != nil {
				return err
			}

			testEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Test"
			})

			if len(testEnvironment) != 1 {
				return errors.New("space must have an environment called \"Test\" in space " + recreatedSpaceId)
			}

			developmentEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Development"
			})

			if len(developmentEnvironment) != 1 {
				return errors.New("space must have an environment called \"Development\" in space " + recreatedSpaceId)
			}

			resourceName := "Test"

			project := lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
				return item.Name == resourceName
			})

			if len(project) != 1 {
				return errors.New("space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
			err = octopusClient.GetAllResources("Projects/"+project[0].Id+"/Triggers", &triggers)

			if err != nil {
				return err
			}

			deployNew := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Deploy New"
			})

			if len(deployNew) != 1 {
				return errors.New("space must have an trigger called \"Deploy New\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(deployNew[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Deploy New\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if strutil.EmptyIfNil(deployNew[0].Action.EnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Deploy New\" must have an environment of Test")
			}

			runbook := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Runbook"
			})

			if len(runbook) != 1 {
				return errors.New("space must have an trigger called \"Runbook\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(runbook[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Runbook\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, testEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Test")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, developmentEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Development")
			}

			return nil
		})
}

// TestSingleProjectLookupScheduledTriggerExport verifies that a project with lookups to external resources
// can be reimported with feed triggers. We defer to the TestProjectScheduledTriggerExport test to verify
// that triggers are created with the correct values. This test is focused on ensuring environments are
// correctly resolved with lookups.
func TestSingleProjectLookupScheduledTriggerExport(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/72-projecttriggerlookup/space_creation",
		"../test/terraform/72-projecttriggerlookup/space_prepopulation",
		"../test/terraform/72-projecttriggerlookup/space_population",
		"../test/terraform/72-projecttriggerlookup/space_creation",
		"../test/terraform/72-projecttriggerlookup/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			ProjectName: []string{"Test"},
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			environments := octopus.GeneralCollection[octopus.Environment]{}
			err = octopusClient.GetAllResources("Environments", &environments)

			if err != nil {
				return err
			}

			testEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Test"
			})

			if len(testEnvironment) != 1 {
				return errors.New("space must have an environment called \"Test\" in space " + recreatedSpaceId)
			}

			developmentEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Development"
			})

			if len(developmentEnvironment) != 1 {
				return errors.New("space must have an environment called \"Development\" in space " + recreatedSpaceId)
			}

			resourceName := "Test"

			project := lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
				return item.Name == resourceName
			})

			if len(project) != 1 {
				return errors.New("space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
			err = octopusClient.GetAllResources("Projects/"+project[0].Id+"/Triggers", &triggers)

			if err != nil {
				return err
			}

			deployNew := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Deploy New"
			})

			if len(deployNew) != 1 {
				return errors.New("space must have an trigger called \"Deploy New\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(deployNew[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Deploy New\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if strutil.EmptyIfNil(deployNew[0].Action.EnvironmentId) != testEnvironment[0].Id {
				return errors.New("the trigger \"Deploy New\" must have an environment of Test")
			}

			runbook := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "Runbook"
			})

			if len(runbook) != 1 {
				return errors.New("space must have an trigger called \"Runbook\" in space " + recreatedSpaceId)
			}

			if strutil.EmptyIfNil(runbook[0].Filter.CronExpression) != "0 0 06 * * Mon-Fri" {
				return errors.New("the trigger \"Runbook\" must have a cron expression of \"0 0 06 * * Mon-Fri\"")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, testEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Test")
			}

			if slices.Index(runbook[0].Action.EnvironmentIds, developmentEnvironment[0].Id) == -1 {
				return errors.New("the trigger \"Runbook\" must have a deployment environment of Development")
			}

			return nil
		})
}

func TestSingleProjectFeedTriggerExport(t *testing.T) {
	exportProjectImportAndTest(
		t,
		"Test",
		"../test/terraform/73-projectfeedtrigger/space_creation",
		"../test/terraform/73-projectfeedtrigger/space_population",
		"../test/terraform/z-createspace",
		[]string{},
		[]string{},
		[]string{
			"-var=feed_docker_password=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			environments := octopus.GeneralCollection[octopus.Environment]{}
			err = octopusClient.GetAllResources("Environments", &environments)

			if err != nil {
				return err
			}

			developmentEnvironment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
				return item.Name == "Development"
			})

			if len(developmentEnvironment) != 1 {
				return errors.New("space must have an environment called \"Development\" in space " + recreatedSpaceId)
			}

			resourceName := "Test"

			project := lo.Filter(collection.Items, func(item octopus.Project, index int) bool {
				return item.Name == resourceName
			})

			if len(project) != 1 {
				return errors.New("space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			triggers := octopus.GeneralCollection[octopus.ProjectTrigger]{}
			err = octopusClient.GetAllResources("Projects/"+project[0].Id+"/Triggers", &triggers)

			if err != nil {
				return err
			}

			deployNew := lo.Filter(triggers.Items, func(item octopus.ProjectTrigger, index int) bool {
				return item.Name == "My feed trigger"
			})

			if len(deployNew) != 1 {
				return errors.New("space must have an trigger called \"My feed trigger\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestGitDependenciesExport verifies that a project can be reimported with steps that have git dependencies
func TestGitDependenciesExport(t *testing.T) {
	exportSpaceImportAndTest(
		t,
		"../test/terraform/74-gitdependencies/space_creation",
		"../test/terraform/74-gitdependencies/space_population",
		[]string{},
		[]string{
			"-var=gitcredential_test=whatever",
		},
		args2.Arguments{},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := octopus.GeneralCollection[octopus.Project]{}
			err := octopusClient.GetAllResources("Projects", &collection)

			if err != nil {
				return err
			}

			resourceName := "Test"
			found := false
			for _, v := range collection.Items {
				if v.Name == resourceName {
					found = true

					deploymentProcess := octopus.DeploymentProcess{}
					_, err := octopusClient.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(v.DeploymentProcessId), &deploymentProcess)

					if err != nil {
						return err
					}

					gitCreds := octopus.GeneralCollection[octopus.GitCredentials]{}
					err = octopusClient.GetAllResources("Git-Credentials", &gitCreds)

					if err != nil {
						return err
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].GitDependencies[0].RepositoryUri) != "https://github.com/OctopusDeploy/OctopusClients.git" {
						return errors.New("step must have git credentials url of \"https://github.com/OctopusDeploy/OctopusClients.git\"")
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].GitDependencies[0].GitCredentialType) != "Library" {
						return errors.New("step must have git credentials type of \"Library\"")
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].GitDependencies[0].DefaultBranch) != "main" {
						return errors.New("step must have git credentials branch of \"main\"")
					}

					if strutil.EmptyIfNil(deploymentProcess.Steps[0].Actions[0].GitDependencies[0].GitCredentialId) != gitCreds.Items[0].Id {
						return errors.New("step must have git credentials ID referencing the git credentials resource")
					}
				}
			}

			if !found {
				return errors.New("space must have an project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}

// TestProjectTenantLinks verifies that a project can recreate tenant links and variables to an existing tenant
func TestProjectTenantLinks(t *testing.T) {
	exportProjectLookupImportAndTest(
		t,
		"Test",
		"../test/terraform/75-projectjointenants/space_creation",
		"../test/terraform/75-projectjointenants/space_prepopulation",
		"../test/terraform/75-projectjointenants/space_population",
		"../test/terraform/75-projectjointenants/space_creation",
		"../test/terraform/75-projectjointenants/space_prepopulation",
		[]string{},
		[]string{},
		[]string{},
		[]string{},
		args2.Arguments{
			LookupProjectLinkTenants: true,
		},
		func(t *testing.T, container *test.OctopusContainer, recreatedSpaceId string, terraformStateDir string) error {

			// Assert
			octopusClient := createClient(container, recreatedSpaceId)

			collection := []octopus.TenantVariable{}
			if err := octopusClient.GetAllResources("TenantVariables/All", &collection); err != nil {
				return err
			}

			environments := octopus.GeneralCollection[octopus.Environment]{}
			if err := octopusClient.GetAllResources("Environments", &environments); err != nil {
				return err
			}

			resourceName := "Test"
			foundProjectVar := false
			foundCommonVar := false
			for _, tenantVariable := range collection {
				for _, project := range tenantVariable.ProjectVariables {
					if project.ProjectName == resourceName {
						for environmentId, variables := range project.Variables {
							for _, value := range variables {

								environment := lo.Filter(environments.Items, func(item octopus.Environment, index int) bool {
									return item.Id == environmentId
								})

								if environment[0].Name == "Development" {
									foundProjectVar = true
									if value != "my project variable" {
										return errors.New("The tenant project variable must have a value of \"my project variable\" (was \"" + value + "\")")
									}
								}
							}
						}
					}
				}

				for _, commonVariables := range tenantVariable.LibraryVariables {
					if commonVariables.LibraryVariableSetName == "Octopus Variables" {
						for _, variable := range commonVariables.Variables {
							if stringVariable, ok := variable.(string); ok {
								foundCommonVar = true
								if stringVariable != "my common variable" {
									return errors.New("The tenant common variable must have a value of \"my common variable\" (was \"" + stringVariable + "\")")
								}
							}
						}
					}
				}
			}

			if !foundProjectVar {
				return errors.New("Space must have an tenant project variable for the project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			if !foundCommonVar {
				return errors.New("Space must have an tenant common variable for the project called \"" + resourceName + "\" in space " + recreatedSpaceId)
			}

			return nil
		})
}
