package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/regexes"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/strings/slices"
	"net/url"
	"strings"
)

const octopusdeployVariableResourceType = "octopusdeploy_variable"

// VariableSetConverter exports variable sets.
// Note that we only access variable sets as dependencies of other resources, like project variables or
// library variable sets. There is no global collection or all endpoint that we can use to dump variables
// in bulk.
type VariableSetConverter struct {
	Client                            client.OctopusClient
	ChannelConverter                  ConverterByProjectIdWithTerraDependencies
	EnvironmentConverter              ConverterAndLookupWithStatelessById
	TagSetConverter                   ConvertToHclByResource[octopus.TagSet]
	AzureCloudServiceTargetConverter  ConverterAndLookupWithStatelessById
	AzureServiceFabricTargetConverter ConverterAndLookupWithStatelessById
	AzureWebAppTargetConverter        ConverterAndLookupWithStatelessById
	CloudRegionTargetConverter        ConverterAndLookupWithStatelessById
	KubernetesTargetConverter         ConverterAndLookupWithStatelessById
	ListeningTargetConverter          ConverterAndLookupWithStatelessById
	OfflineDropTargetConverter        ConverterAndLookupWithStatelessById
	PollingTargetConverter            ConverterAndLookupWithStatelessById
	SshTargetConverter                ConverterAndLookupWithStatelessById
	AccountConverter                  ConverterAndLookupWithStatelessById
	FeedConverter                     ConverterAndLookupWithStatelessById
	CertificateConverter              ConverterAndLookupWithStatelessById
	WorkerPoolConverter               ConverterAndLookupWithStatelessById
	IgnoreCacManagedValues            bool
	DefaultSecretVariableValues       bool
	DummySecretVariableValues         bool
	ExcludeAllProjectVariables        bool
	ExcludeProjectVariables           args.StringSliceArgs
	ExcludeProjectVariablesExcept     args.StringSliceArgs
	ExcludeProjectVariablesRegex      args.StringSliceArgs
	ExcludeTenantTagSets              args.StringSliceArgs
	ExcludeTenantTags                 args.StringSliceArgs
	IgnoreProjectChanges              bool
	DummySecretGenerator              dummy.DummySecretGenerator

	Excluder                  ExcludeByName
	ErrGroup                  *errgroup.Group
	ExcludeTerraformVariables bool
	LimitAttributeLength      int
	StatelessAdditionalParams args.StringSliceArgs
	GenerateImportScripts     bool
	EnvironmentFilter         EnvironmentFilter
}

func (c *VariableSetConverter) ToHclByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdBranchAndName(projectId, branch, parentName, parentLookup, parentCount, recursive, false, dependencies)
}

func (c *VariableSetConverter) ToHclStatelessByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdBranchAndName(projectId, branch, parentName, parentLookup, parentCount, recursive, true, dependencies)
}

func (c *VariableSetConverter) toHclByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResource("Projects/"+projectId+"/"+url.QueryEscape(branch)+"/variables", &resource)

	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, recursive, false, stateless, ignoreSecrets, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) ToHclLookupByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResource("Projects/"+projectId+"/"+url.QueryEscape(branch)+"/variables", &resource)

	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, ignoreSecrets, parentName, parentLookup, nil, dependencies)
}

// ToHclByProjectIdAndName is called when returning variables from projects. This is because the variable set ID
// defined on a CaC enabled project is not available from the global /variablesets endpoint, and can only be
// accessed from the project resource.
func (c *VariableSetConverter) ToHclByProjectIdAndName(projectId string, parentName string, parentLookup string, parentCount *string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &resource)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, recursive, false, parentCount != nil, ignoreSecrets, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) ToHclLookupByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	_, err := c.Client.GetResource(c.GetGroupResourceType(projectId), &resource)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, ignoreSecrets, parentName, parentLookup, nil, dependencies)
}

func (c *VariableSetConverter) ToHclByIdAndName(id string, recursive bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, recursive, false, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) ToHclStatelessByIdAndName(id string, recursive bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, recursive, true, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) toHclByIdAndName(id string, recursive bool, stateless bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Some CaC enabled projects have no variable set.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, recursive, false, stateless, false, parentName, parentLookup, parentCount, dependencies)
}

// ToHclLookupByIdAndName exports the variable set as a complete resource, but will reference external resources like accounts,
// feeds, worker pools, certificates, environments, and targets as data source lookups.
func (c *VariableSetConverter) ToHclLookupByIdAndName(id string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Some CaC enabled projects have no variable set.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, false, parentName, parentLookup, nil, dependencies)
}

// toProjectPowershellImport creates a powershell script to import the resource from an existing project
func (c *VariableSetConverter) toProjectPowershellImport(resourceName string, octopusProjectName string, octopusResourceName string, envNames []string, machineNames []string, roleNames []string, channelNames []string, actionNames []string, ownerNames []string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_project_variable_" + resourceName + ".ps1",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.ps1 API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

param (
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,

    [Parameter(Mandatory=$true)]
    [string]$Url,

    [Parameter(Mandatory=$true)]
    [string]$SpaceId
)

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$EnvScopes="%s".Split(",")
$MachineScopes="%s".Split(",")
$RoleScopes="%s".Split(",")
$ChannelScopes="%s".Split(",")
$ActionScopes="%s".Split(",")
$OwnerScopes="%s".Split(",")

$ProjectName="%s"

$ProjectId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ProjectName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ProjectName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ProjectId)) {
	echo "No project found with the name $ProjectName"
	exit 1
}

$ResourceName="%s"

$Variables = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/Variables" -Method Get -Headers $headers
$DeploymentProcess = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectId/DeploymentProcesses" -Method Get -Headers $headers

$Resource = $Variables |
	Select-Object -ExpandProperty Variables | 
	Where-Object {$_.Name -eq $ResourceName}

if ($Resource -eq $null) {
	echo "No variable found with the name $ResourceName"
	exit 1
}

function Test-ArraysEqual {
	param(
        [Parameter(Mandatory=$false)]
        [string[]]$array1,

        [Parameter(Mandatory=$false)]
        [string[]]$array2
    )

	if ($array1 -eq $null) {
		$array1 = @()
	}

	if ($array2 -eq $null) {
		$array2 = @()
	}

	# Sort the arrays
	$sortedArray1 = $array1 | Sort-Object
	$sortedArray2 = $array2 | Sort-Object
	
	if ($sortedArray1 -eq $null) {
		$sortedArray1 = @()
	}

	if ($sortedArray2 -eq $null) {
		$sortedArray2 = @()
	}
	
	Write-Host "Comparing Arrays"
	Write-Host "Destination Variable Scopes: $($sortedArray1 -join ",")"
	Write-Host "Source Variable Scopes: $($sortedArray2 -join ",")"

	if ($sortedArray1.Length -eq 0 -and $sortedArray2.Length -eq 0) {
		return $True
	}
	
	# Compare the sorted arrays
	$result = Compare-Object -ReferenceObject $sortedArray1 -DifferenceObject $sortedArray2
	return -not $result
}

function Get-ProjectName {
	param(
        [Parameter(Mandatory=$false)]
        [string[]]$ProjectOwner
    )

	$Project = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$ProjectOwner" -Method Get -Headers $headers
	return $Project.Name
}

function Get-RunbookName {
	param(
        [Parameter(Mandatory=$false)]
        [string[]]$RunbookOwner
    )

	$Runbook = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Runbooks/$RunbookOwner" -Method Get -Headers $headers
	$Project = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects/$Runbook.ProjectId" -Method Get -Headers $headers
	return $Project.Name + ":" + $Runbook.Name
}

# Check environment scopes
echo "Testing environments"
$Resource = $Resource | Where-Object { 
	$ScopedEnvironments = $_.Scope.Environment | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Environments | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedEnvironments $EnvScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same environment scopes"
	exit 1
}

# Check machine scopes
echo "Testing machines"
$Resource = $Resource | Where-Object { 
	$ScopedMachines = $_.Scope.Machine | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Machines | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedMachines $MachineScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same machine scopes"
	exit 1
}

# Check role scopes
echo "Testing roles"
$Resource = $Resource | Where-Object { Test-ArraysEqual $_.Scope.Role $RoleScopes }

if ($Resource.Count -eq 0) {
	echo "No variable found with the same role scopes"
	exit 1
}

# Check channel scopes
echo "Testing channels"
$Resource = $Resource | Where-Object { 
	$ScopedChannels = $_.Scope.Channel | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Channels | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedChannels $ChannelScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same channel scopes"
	exit 1
}

# Check action scopes
echo "Testing actions"
$Resource = $Resource | Where-Object { 
	$ScopedActions = $_.Scope.Action | ForEach-Object {$ActionId = $_; $DeploymentProcess.Steps | ForEach-Object {$_.Actions} | Where-Object {$ActionId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedActions $ActionScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same action scopes"
	exit 1
}

# Check owner scopes
echo "Testing owners"
$Resource = $Resource | Where-Object { 
	$ScopedOwners = $_.Scope.ProcessOwner | ForEach-Object {
		if ([string]::IsNullOrWhiteSpace($_)) {
			return $null
		}
	
		if ($_.StartsWith("Projects")) {
			return Get-ProjectName $_
		}

		if ($_.StartsWith("Runbooks")) {
			return Get-RunbookName $_
		}

		return $null
	} | Where-Object {$_ -ne $null}

	Test-ArraysEqual $ScopedOwners $OwnerScopes
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same process owner scopes"
	exit 1
}

$ResourceId = $Resource.Id
echo "Importing variable $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s "$($ProjectId):$($ResourceId)"`,
				resourceName,
				strings.Join(envNames, ","),
				strings.Join(machineNames, ","),
				strings.Join(roleNames, ","),
				strings.Join(channelNames, ","),
				strings.Join(actionNames, ","),
				strings.Join(ownerNames, ","),
				octopusProjectName,
				octopusResourceName,
				octopusdeployVariableResourceType,
				resourceName), nil
		},
	})
}

// toVariableSetPowershellImport creates a powershell script to import the resource from an existing variable set
func (c *VariableSetConverter) toVariableSetPowershellImport(resourceName string, octopusProjectName string, octopusResourceName string, envNames []string, machineNames []string, roleNames []string, channelNames []string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_library_variable_set_variable_" + resourceName + ".ps1",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.ps1 API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

param (
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,

    [Parameter(Mandatory=$true)]
    [string]$Url,

    [Parameter(Mandatory=$true)]
    [string]$SpaceId
)

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$EnvScopes="%s".Split(",")
$MachineScopes="%s".Split(",")
$RoleScopes="%s".Split(",")
$ChannelScopes="%s".Split(",")

$VariableSetName="%s"

$VariableSet = Invoke-RestMethod -Uri "$Url/api/$SpaceId/LibraryVariableSets?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($VariableSetName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $VariableSetName} 

if ($VariableSet -eq $null) {
	echo "No library variable set found with the name $VariableSetName"
	exit 1
}

$VariableSetId = $VariableSet.Id

$ResourceName="%s"

$Variables = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Variables/$($VariableSet.VariableSetId)" -Method Get -Headers $headers

$Resource = $Variables |
	Select-Object -ExpandProperty Variables | 
	Where-Object {$_.Name -eq $ResourceName}

if ($Resource -eq $null) {
	echo "No variable found with the name $ResourceName"
	exit 1
}

function Test-ArraysEqual {
	param(
        [Parameter(Mandatory=$false)]
        [string[]]$array1,

        [Parameter(Mandatory=$false)]
        [string[]]$array2
    )

	if ($array1 -eq $null) {
		$array1 = @()
	}

	if ($array2 -eq $null) {
		$array2 = @()
	}

	# Sort the arrays
	$sortedArray1 = $array1 | Sort-Object
	$sortedArray2 = $array2 | Sort-Object
	
	if ($sortedArray1 -eq $null) {
		$sortedArray1 = @()
	}

	if ($sortedArray2 -eq $null) {
		$sortedArray2 = @()
	}
	
	Write-Host "Comparing Arrays"
	Write-Host "Destination Variable Scopes: $($sortedArray1 -join ",")"
	Write-Host "Source Variable Scopes: $($sortedArray2 -join ",")"

	if ($sortedArray1.Length -eq 0 -and $sortedArray2.Length -eq 0) {
		return $True
	}
	
	# Compare the sorted arrays
	$result = Compare-Object -ReferenceObject $sortedArray1 -DifferenceObject $sortedArray2
	return -not $result
}

# Check environment scopes
echo "Testing environments"
$Resource = $Resource | Where-Object { 
	$ScopedEnvironments = $_.Scope.Environment | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Environments | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedEnvironments $EnvScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same environment scopes"
	exit 1
}

# Check machine scopes
echo "Testing machines"
$Resource = $Resource | Where-Object { 
	$ScopedMachines = $_.Scope.Machine | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Machines | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedMachines $MachineScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same machine scopes"
	exit 1
}

# Check role scopes
echo "Testing roles"
$Resource = $Resource | Where-Object { Test-ArraysEqual $_.Scope.Role $RoleScopes }

if ($Resource.Count -eq 0) {
	echo "No variable found with the same role scopes"
	exit 1
}

# Check channel scopes
echo "Testing channels"
$Resource = $Resource | Where-Object { 
	$ScopedChannels = $_.Scope.Channel | ForEach-Object {$EnvId = $_; $Variables.ScopeValues.Channels | Where-Object{$EnvId -eq $_.Id} | Select-Object -ExpandProperty Name}
	Test-ArraysEqual $ScopedChannels $ChannelScopes 
}

if ($Resource.Count -eq 0) {
	echo "No variable found with the same channel scopes"
	exit 1
}

$ResourceId = $Resource.Id
echo "Importing variable $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s "$($VariableSetId):$($ResourceId)"`,
				resourceName,
				strings.Join(envNames, ","),
				strings.Join(machineNames, ","),
				strings.Join(roleNames, ","),
				strings.Join(channelNames, ","),
				octopusProjectName,
				octopusResourceName,
				octopusdeployVariableResourceType,
				resourceName), nil
		},
	})
}

// processImportScript converts all the variable scopes from IDs back to names and passes the scope names to the
// scripts used to import existing variables. It takes care of the differences between project and library variable sets
// variables, where the later have more limited scoping options.
func (c *VariableSetConverter) processImportScript(resourceName string, parentId string, v octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if !c.GenerateImportScripts {
		return nil
	}

	// Build the import script
	scopedEnvironmentNames, lookupErrors := c.Client.GetResourceNamesByIds("Environments", c.EnvironmentFilter.FilterEnvironmentScope(v.Scope.Environment))

	if lookupErrors != nil {
		return lookupErrors
	}

	scopedMachineNames, lookupErrors := c.Client.GetResourceNamesByIds("Machines", v.Scope.Machine)

	if lookupErrors != nil {
		return lookupErrors
	}

	scopedChannelNames, lookupErrors := c.Client.GetResourceNamesByIds("Channel", v.Scope.Channel)

	if lookupErrors != nil {
		return lookupErrors
	}

	if strings.HasPrefix(parentId, "Projects") {
		project := octopus.Project{}
		_, projectErr := c.Client.GetSpaceResourceById("Projects", parentId, &project)

		if projectErr != nil {
			return projectErr
		}

		// Only variables assigned to a project can be scoped to owners
		var ownersError error = nil
		scopedOwners := lo.Map(v.Scope.ProcessOwner, func(owner string, index int) string {
			if strings.HasPrefix(owner, "Projects") {
				projectName, err := c.Client.GetResourceNameById("Projects", owner)

				if err != nil {
					ownersError = errors.Join(ownersError, err)
				}

				return projectName
			}

			if strings.HasPrefix(owner, "Runbooks") {
				runbook := octopus.Runbook{}
				_, runbookErr := c.Client.GetSpaceResourceById("Runbooks", owner, &runbook)

				if runbookErr != nil {
					ownersError = errors.Join(ownersError, runbookErr)
				} else {

					project := octopus.Project{}
					_, projectErr := c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

					if projectErr != nil {
						ownersError = errors.Join(ownersError, projectErr)
					}

					return project.Name + ":" + runbook.Name
				}
			}

			ownersError = errors.Join(ownersError, errors.New("Found unexpected owner with ID "+owner))
			return ""
		})

		if ownersError != nil {
			return ownersError
		}

		scopedActions := []string{}

		// Only variables assigned to a project can be scoped to individual actions
		if project.DeploymentProcessId != nil {
			deploymentProcess := octopus.DeploymentProcess{}
			_, processErr := c.Client.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(project.DeploymentProcessId), &deploymentProcess)

			if processErr != nil {
				return processErr
			}

			scopedActions = lo.FilterMap(v.Scope.Action, func(actionId string, index int) (string, bool) {
				actions := lo.FlatMap(deploymentProcess.Steps, func(step octopus.Step, index int) []octopus.Action {
					return step.Actions
				})
				step := lo.Filter(actions, func(action octopus.Action, index int) bool {
					return actionId == action.Id
				})

				if len(step) == 0 {
					return "", false
				}

				return strutil.EmptyIfNil(step[0].Name), true
			})
		}

		c.toProjectPowershellImport(
			resourceName,
			project.Name,
			v.Name,
			scopedEnvironmentNames,
			scopedMachineNames,
			v.Scope.Role,
			scopedChannelNames,
			scopedActions,
			scopedOwners,
			dependencies)
	} else if strings.HasPrefix(parentId, "LibraryVariableSets") {
		libraryVariableSetName, lookupErr := c.Client.GetResourceNameById("LibraryVariableSets", parentId)

		if lookupErr != nil {
			return lookupErr
		}

		c.toVariableSetPowershellImport(
			resourceName,
			libraryVariableSetName,
			v.Name,
			scopedEnvironmentNames,
			scopedMachineNames,
			v.Scope.Role,
			scopedChannelNames,
			dependencies)

	}

	return nil
}

func (c *VariableSetConverter) toHcl(resource octopus.VariableSet, recursive bool, lookup bool, stateless bool, ignoreSecrets bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	nameCount := map[string]int{}
	for _, v := range resource.Variables {
		// Do not export regular variables if ignoring cac managed values
		if ignoreSecrets && !v.IsSensitive {
			continue
		}

		// Do not export excluded variables
		if c.Excluder.IsResourceExcludedWithRegex(v.Name, c.ExcludeAllProjectVariables, c.ExcludeProjectVariables, c.ExcludeProjectVariablesRegex, c.ExcludeProjectVariablesExcept) {
			continue
		}

		// Generate a unique suffix for each variable name
		if count, ok := nameCount[v.Name]; ok {
			nameCount[v.Name] = count + 1
		} else {
			nameCount[v.Name] = 1
		}

		v := v
		file := hclwrite.NewEmptyFile()
		thisResource := data.ResourceDetails{}

		resourceName := sanitizer.SanitizeName(parentName) + "_" + sanitizer.SanitizeName(v.Name) + "_" + fmt.Sprint(nameCount[v.Name])

		if err := c.processImportScript(resourceName, strutil.EmptyIfNil(resource.OwnerId), v, dependencies); err != nil {
			return err
		}

		// Export linked accounts
		err := c.exportAccounts(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked feeds
		err = c.exportFeeds(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked worker pools
		err = c.exportWorkerPools(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked certificates
		err = c.exportCertificates(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked environments
		err = c.exportEnvironments(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure cloud service targets
		err = c.exportAzureCloudServiceTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure service fabric targets
		err = c.exportAzureServiceFabricTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportAzureWebAppTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportCloudRegionTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export kubernetes targets
		err = c.exportKubernetesTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportListeningTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportOfflineDropTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportPollingTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportSshTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Placing sensitive variables in uniquely prefixed files allows us to target them for variable substitution
		if v.IsSensitive {
			thisResource.FileName = "space_population/project_variable_sensitive_" + resourceName + ".tf"
		} else {
			thisResource.FileName = "space_population/project_variable_" + resourceName + ".tf"
		}

		thisResource.Id = v.Id
		thisResource.Name = v.Name
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${" + octopusdeployVariableResourceType + "." + resourceName + ".id}"
		if v.IsSensitive {
			thisResource.Parameters = []data.ResourceParameter{
				{
					Label: "Sensitive variable " + v.Name + " password",
					Description: "The sensitive value associated with the variable \"" + v.Name + "\" belonging to " +
						parentName + v.Scope.ScopeDescription(" (", ")", dependencies),
					ResourceName:  sanitizer.SanitizeParameterName(dependencies, v.Name, "SensitiveValue"),
					Sensitive:     true,
					VariableName:  resourceName,
					ParameterType: "SensitiveValue",
				},
			}
		} else if slices.Contains(c.StatelessAdditionalParams, parentName+":"+v.Name) {
			thisResource.Parameters = []data.ResourceParameter{
				{
					Label: "Variable " + v.Name + " value",
					Description: "The value associated with the variable \"" + v.Name + "\" belonging to " +
						parentName + v.Scope.ScopeDescription(" (", ")", dependencies),
					ResourceName:  sanitizer.SanitizeParameterName(dependencies, v.Name, "SensitiveValue"),
					Sensitive:     false,
					VariableName:  resourceName,
					ParameterType: "SingleLineText",
					DefaultValue:  strutil.EscapeDollarCurly(strutil.EmptyIfNil(v.Value)),
				},
			}
		}
		thisResource.ToHcl = func() (string, error) {

			// Replace anything that looks like an octopus resource reference
			value := strutil.EscapeDollarCurlyPointer(v.Value)
			value = c.getAccount(value, dependencies)
			value = c.getFeeds(value, dependencies)
			value = c.getCertificates(value, dependencies)
			value = c.getWorkerPools(value, dependencies)

			normalValue := c.writeTerraformVariablesForString(file, v, resourceName, value)
			sensitiveValue := c.writeTerraformVariablesForSecret(file, v, resourceName, dependencies)

			terraformResource := terraform.TerraformProjectVariable{
				Name:           resourceName,
				Type:           octopusdeployVariableResourceType,
				Count:          parentCount,
				OwnerId:        parentLookup,
				Value:          normalValue,
				ResourceName:   v.Name,
				ResourceType:   v.Type,
				Description:    v.Description,
				SensitiveValue: nil,
				IsSensitive:    v.IsSensitive,
				Prompt:         c.convertPrompt(v.Prompt),
				Scope:          c.convertScope(v, dependencies),
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if sensitiveValue != nil {
				hcl.WriteUnquotedAttribute(block, "sensitive_value", strutil.EmptyIfNil(sensitiveValue))
			}

			if c.IgnoreProjectChanges || c.DummySecretVariableValues || stateless {
				ignoreAll := terraform.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				block.Body().AppendBlock(lifecycleBlock)

				if c.IgnoreProjectChanges {
					// Ignore all changes if requested
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "all")
				} else if c.DummySecretVariableValues && !c.DefaultSecretVariableValues {
					// When using dummy values, and not using default secret values, we expect the secrets will be updated later
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[sensitive_value]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			// If we are creating the tag sets (i.e. exporting a space or recursively exporting a project),
			// ensure tag sets are create before the variable.
			// If we are doing a lookup, the tag sets are expected to already be available, and so there is
			// no dependency relationship.
			if !lookup {
				tagSetDependencies, err := c.addTagSetDependencies(v, recursive, dependencies)

				if err != nil {
					return "", err
				}

				// Explicitly describe the dependency between a variable and a tag set
				dependsOn := []string{}
				for resourceType, terraformDependencies := range tagSetDependencies {
					for _, terraformDependency := range terraformDependencies {
						dependency := dependencies.GetResourceDependency(resourceType, terraformDependency)
						dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
						dependsOn = append(dependsOn, dependency)
					}
				}
				hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")
			}

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c *VariableSetConverter) writeTerraformVariablesForSecret(file *hclwrite.File, variable octopus.Variable, resourceName string, dependencies *data.ResourceDetailsCollection) *string {
	if variable.IsSensitive {
		// We don't know the value of secrets, so the value is just nil
		if c.ExcludeTerraformVariables {
			return nil
		}

		var defaultValue *string = nil

		// Dummy values are used if we are not also replacing the variable with a octostache template
		// with the DefaultSecretVariableValues option.
		if c.DummySecretVariableValues && !c.DefaultSecretVariableValues {
			defaultValue = c.DummySecretGenerator.GetDummySecret()
			dependencies.AddDummy(data.DummyVariableReference{
				VariableName: resourceName,
				ResourceName: variable.Name,
				ResourceType: c.GetResourceType(),
			})
		}

		secretVariableResource := terraform.TerraformVariable{
			Name:        resourceName,
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The secret variable value associated with the variable " + variable.Name,
			Default:     defaultValue,
		}

		block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")

		// If we are writing an octostache template, we need to have any string escaped for inclusion in a terraform
		// string. JSON escaping will get us most of the way there. We also need to escape any terraform syntax, which
		// unfortunately is easier said than done as there appears to be no way to write a double dollar sign with
		// the HCL serialization library, so we need to get a little creative.
		if c.DefaultSecretVariableValues {
			hcl.WriteUnquotedAttribute(block, "default", "<<EOT\n#{"+variable.Name+" | Replace \"([$])([{])\" \"$1$1$2\" | Replace \"([%])([{])\" \"$1$1$2\"}\nEOT")
		}

		file.Body().AppendBlock(block)

		return c.convertSecretValue(variable, resourceName)
	}

	return nil
}

func (c *VariableSetConverter) writeTerraformVariablesForString(file *hclwrite.File, variable octopus.Variable, resourceName string, value *string) *string {
	if c.ExcludeTerraformVariables {
		return value
	}

	if variable.Type == "String" && !hcl.IsInterpolation(strutil.EmptyIfNil(value)) {
		// Use a second terraform variable to allow the octopus variable to be defined at apply time.
		// Note this only applies to string variables, as other types likely reference resources
		// that are being created by terraform, and these dynamic values can not be used as default
		// variable values.

		regularVariable := terraform.TerraformVariable{
			Name:        resourceName,
			Type:        "string",
			Nullable:    true,
			Sensitive:   false,
			Description: "The value associated with the variable " + variable.Name,
			Default:     strutil.StrPointer(LimitAttributeLength(c.LimitAttributeLength, true, strutil.EmptyIfNil(value))),
		}

		block := gohcl.EncodeAsBlock(regularVariable, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return c.convertValue(variable, resourceName)
	}

	return value
}

func (c *VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c *VariableSetConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Variables"
}

func (c *VariableSetConverter) convertSecretValue(variable octopus.Variable, resourceName string) *string {
	if !variable.IsSensitive {
		return nil
	}

	// The heredoc string introduces a line break at the end of the string. We remove it here.
	// See https://discuss.hashicorp.com/t/trailing-new-line-in-key-vault-after-using-heredoc-syntax/14561
	if c.DefaultSecretVariableValues {
		value := "replace(var." + resourceName + ", \"\\n$\", \"\")"
		return &value
	}

	value := "var." + resourceName
	return &value
}

func (c *VariableSetConverter) convertValue(variable octopus.Variable, resourceName string) *string {
	if !variable.IsSensitive {
		value := "${var." + resourceName + "}"
		return &value
	}

	return nil
}

func (c *VariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if strutil.EmptyIfNil(prompt.Label) != "" || strutil.EmptyIfNil(prompt.Description) != "" {
		return &terraform.TerraformProjectVariablePrompt{
			Description:     prompt.Description,
			Label:           prompt.Label,
			IsRequired:      prompt.Required,
			DisplaySettings: c.convertDisplaySettings(prompt),
		}
	}

	return nil
}

func (c *VariableSetConverter) convertDisplaySettings(prompt octopus.Prompt) *terraform.TerraformProjectVariableDisplay {
	if prompt.DisplaySettings == nil || len(prompt.DisplaySettings) == 0 {
		return nil
	}

	display := terraform.TerraformProjectVariableDisplay{}
	if controlType, ok := prompt.DisplaySettings["Octopus.ControlType"]; ok {
		display.ControlType = strutil.StrPointer("SingleLineText")

		// The provider only recognises the following options. Notably, it does not recognise "Sensitive".
		// We do our best, but fall back to "SingleLineText".
		if slices.Index([]string{"SingleLineText", "MultiLineText", "Checkbox", "Select"}, controlType) != -1 {
			display.ControlType = &controlType
		}
	}

	selectOptionsSlice := []terraform.TerraformProjectVariableDisplaySelectOption{}
	if selectOptions, ok := prompt.DisplaySettings["Octopus.SelectOptions"]; ok {
		for _, o := range strings.Split(selectOptions, "\n") {
			split := strings.Split(o, "|")
			if len(split) == 2 {
				selectOptionsSlice = append(
					selectOptionsSlice,
					terraform.TerraformProjectVariableDisplaySelectOption{
						DisplayName: split[0],
						Value:       split[1],
					})
			}
		}
	}
	display.SelectOption = &selectOptionsSlice

	return &display
}

func (c *VariableSetConverter) convertScope(variable octopus.Variable, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProjectVariableScope {
	filteredEnvironments := c.EnvironmentFilter.FilterEnvironmentScope(variable.Scope.Environment)

	// Removing all environment scoping may not have been the intention
	if len(filteredEnvironments) == 0 && len(variable.Scope.Environment) != 0 {
		zap.L().Warn("WARNING: Variable " + variable.Name + " removed all environment scopes.")
	}

	actions := dependencies.GetResources("Actions", variable.Scope.Action...)
	channels := dependencies.GetResources("Channels", variable.Scope.Channel...)
	environments := dependencies.GetResources("Environments", filteredEnvironments...)
	machines := dependencies.GetResources("Machines", variable.Scope.Machine...)

	if len(actions) != 0 ||
		len(channels) != 0 ||
		len(environments) != 0 ||
		len(machines) != 0 ||
		len(variable.Scope.Role) != 0 ||
		len(variable.Scope.TenantTag) != 0 {

		return &terraform.TerraformProjectVariableScope{
			Actions:      actions,
			Channels:     channels,
			Environments: environments,
			Machines:     machines,
			Roles:        variable.Scope.Role,
			TenantTags:   c.Excluder.FilteredTenantTags(variable.Scope.TenantTag, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
		}
	}

	return nil

}

func (c *VariableSetConverter) exportAccounts(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	for _, account := range regexes.AccountRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.AccountConverter.ToHclStatelessById(account, dependencies)
			} else {
				err = c.AccountConverter.ToHclById(account, dependencies)
			}
		} else if lookup {
			err = c.AccountConverter.ToHclLookupById(account, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getAccount(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value

	for _, account := range regexes.AccountRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Accounts", account))
	}

	return &retValue
}

func (c *VariableSetConverter) exportFeeds(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, feed := range regexes.FeedRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.FeedConverter.ToHclStatelessById(feed, dependencies)
			} else {
				err = c.FeedConverter.ToHclById(feed, dependencies)
			}
		} else if lookup {
			err = c.FeedConverter.ToHclLookupById(feed, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getFeeds(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	for _, account := range regexes.FeedRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Feeds", account))
	}

	return &retValue
}

func (c *VariableSetConverter) exportAzureCloudServiceTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureCloudServiceTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureCloudServiceTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureCloudServiceTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportAzureServiceFabricTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureServiceFabricTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureServiceFabricTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureServiceFabricTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportAzureWebAppTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureWebAppTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureWebAppTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureWebAppTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportCloudRegionTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.CloudRegionTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.CloudRegionTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.CloudRegionTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportKubernetesTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.KubernetesTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.KubernetesTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.KubernetesTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportListeningTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.ListeningTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.ListeningTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.ListeningTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportOfflineDropTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.OfflineDropTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.OfflineDropTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.OfflineDropTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportPollingTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.PollingTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.PollingTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.PollingTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportSshTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.SshTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.SshTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.SshTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportEnvironments(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range c.EnvironmentFilter.FilterEnvironmentScope(variable.Scope.Environment) {
		var err error
		if recursive {
			if stateless {
				err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.EnvironmentConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.EnvironmentConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportCertificates(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, cert := range regexes.CertificatesRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.CertificateConverter.ToHclStatelessById(cert, dependencies)
			} else {
				err = c.CertificateConverter.ToHclById(cert, dependencies)
			}
		} else if lookup {
			err = c.CertificateConverter.ToHclLookupById(cert, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getCertificates(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	for _, cert := range regexes.CertificatesRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("Certificates", cert))
	}

	return &retValue
}

func (c *VariableSetConverter) exportWorkerPools(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	for _, pool := range regexes.WorkerPoolsRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.WorkerPoolConverter.ToHclStatelessById(pool, dependencies)
			} else {
				err = c.WorkerPoolConverter.ToHclById(pool, dependencies)
			}
		} else if lookup {
			err = c.WorkerPoolConverter.ToHclLookupById(pool, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getWorkerPools(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if len(strutil.EmptyIfNil(value)) == 0 {
		return nil
	}

	retValue := *value
	for _, cert := range regexes.WorkerPoolsRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("WorkerPools", cert))
	}

	return &retValue
}

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c *VariableSetConverter) addTagSetDependencies(variable octopus.Variable, recursive bool, dependencies *data.ResourceDetailsCollection) (map[string][]string, error) {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := c.Client.GetAllResources("TagSets", &collection)

	if err != nil {
		return nil, err
	}

	terraformDependencies := map[string][]string{}

	for _, tagSet := range collection.Items {
		for _, tag := range tagSet.Tags {
			for _, tenantTag := range variable.Scope.TenantTag {
				if tag.CanonicalTagName == tenantTag {

					if !slices.Contains(terraformDependencies["TagSets"], tagSet.Id) {
						terraformDependencies["TagSets"] = append(terraformDependencies["TagSets"], tagSet.Id)
					}

					if !slices.Contains(terraformDependencies["Tags"], tag.Id) {
						terraformDependencies["Tags"] = append(terraformDependencies["Tags"], tag.Id)
					}

					if recursive {
						err = c.TagSetConverter.ToHclByResource(tagSet, dependencies)

						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return terraformDependencies, nil
}
