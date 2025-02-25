package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/intutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strings"
)

type AccountConverter struct {
	Client                    client.OctopusClient
	EnvironmentConverter      ConverterById
	TenantConverter           ConverterById
	DummySecretVariableValues bool
	DummySecretGenerator      dummy.DummySecretGenerator
	ExcludeTenantTags         args.StringSliceArgs
	ExcludeTenantTagSets      args.StringSliceArgs
	Excluder                  ExcludeByName
	TagSetConverter           ConvertToHclByResource[octopus.TagSet]
	ErrGroup                  *errgroup.Group
	ExcludeAccounts           args.StringSliceArgs
	ExcludeAccountsRegex      args.StringSliceArgs
	ExcludeAccountsExcept     args.StringSliceArgs
	ExcludeAllAccounts        bool
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
}

func (c AccountConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c AccountConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c AccountConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllAccounts {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Account]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllAccounts, c.ExcludeAccounts, c.ExcludeAccountsRegex, c.ExcludeAccountsExcept) {
			continue
		}

		zap.L().Info("Account: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AccountConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c AccountConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c AccountConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Account{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Account: %w", err)
	}

	zap.L().Info("Account: " + resource.Id + " " + resource.Name)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c AccountConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Account{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Account: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllAccounts, c.ExcludeAccounts, c.ExcludeAccountsRegex, c.ExcludeAccountsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "account_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_accounts." + resourceName + ".accounts[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an account called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.accounts) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) buildData(resourceName string, resource octopus.Account) terraform.TerraformAccountData {
	terraformResource := terraform.TerraformAccountData{
		Type:        "octopusdeploy_accounts",
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}

	// Google account types are not defined in the data resource (this is a bug), so don't use it
	if resource.AccountType != "GoogleCloudAccount" {
		terraformResource.AccountType = strutil.StrPointer(resource.AccountType)
	}

	return terraformResource
}

// toBashImport creates a bash script to import the resource
func (c AccountConverter) toBashImport(resourceType string, resourceName string, octopusResourceName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".sh",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`#!/bin/bash

# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Make the script executable with the command:
# chmod +x ./import_%s.sh

# Alternativly, run the script with bash directly:
# /bin/bash ./import_%s.sh <options>

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

if [[ $# -ne 3 ]]
then
	echo "Usage: ./import_%s.sh <API Key> <Octopus URL> <Space ID>"
    echo "Example: ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234"
	exit 1
fi

if ! command -v jq &> /dev/null
then
    echo "jq is required"
    exit 1
fi

if ! command -v curl &> /dev/null
then
    echo "curl is required"
    exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Accounts" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing account ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, octopusResourceName, resourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c AccountConverter) toPowershellImport(resourceType string, resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".ps1",
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

$ResourceName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Accounts?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No account found with the name $ResourceName"
	exit 1
}

echo "Importing account $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, resourceType, resourceName), nil
		},
	})
}

// toHcl adds this resource to the list of dependencies.
// account is the Octopus account object to be serialized
// recursive indicates if any transient dependencies are to be serialized
// stateless indicates if the resource is to be exported for use with a stateless Terraform transaction (i.e. where the
// Terraform state is not maintained between apply commands)
// dependencies maintains the collection of exported Terraform resources
func (c AccountConverter) toHcl(account octopus.Account, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(account.Name, c.ExcludeAllAccounts, c.ExcludeAccounts, c.ExcludeAccountsRegex, c.ExcludeAccountsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + account.Id)
		return nil
	}

	if recursive {
		err := c.exportDependencies(account, dependencies)

		if err != nil {
			return err
		}
	}

	resourceName := "account_" + sanitizer.SanitizeName(account.Name)

	thisResource := data.ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = account.Id
	thisResource.Name = account.Name
	thisResource.ResourceType = c.GetResourceType()

	if account.AccountType == "AmazonWebServicesAccount" {
		c.writeAwsAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_aws_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_aws_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "AzureServicePrincipal" {
		c.writeAzureServicePrincipalAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_azure_service_principal", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_azure_service_principal", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "AzureSubscription" {
		c.writeAzureSubscriptionAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_azure_subscription_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_azure_subscription_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "GoogleCloudAccount" {
		c.writeGoogleCloudAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_gcp_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_gcp_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "Token" {
		c.writeTokenAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_token_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_token_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "UsernamePassword" {
		c.writeUsernamePasswordAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_username_password_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_username_password_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "SshKeyPair" {
		c.writeSshAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_ssh_key_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_ssh_key_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "AmazonWebServicesOidcAccount" {
		err := c.writeAwsOidcAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if err != nil {
			return err
		}

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_aws_openid_connect_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_aws_openid_connect_account", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "AzureOidc" {
		c.writeAzureOidcAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_azure_openid_connect", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_azure_openid_connect", resourceName, account.Name, dependencies)
		}
	} else if account.AccountType == "GenericOidcAccount" {
		c.writeGenericOidcAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)

		if c.GenerateImportScripts {
			c.toBashImport("octopusdeploy_generic_oidc_account", resourceName, account.Name, dependencies)
			c.toPowershellImport("octopusdeploy_generic_oidc_account", resourceName, account.Name, dependencies)
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) GetResourceType() string {
	return "Accounts"
}

func (c AccountConverter) createSecretVariable(resourceName string, description string, accountName string, dependencies *data.ResourceDetailsCollection) terraform.TerraformVariable {
	secretVariableResource := terraform.TerraformVariable{
		Name:        resourceName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: description,
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: resourceName,
			ResourceName: accountName,
			ResourceType: c.GetResourceType(),
		})
	}

	return secretVariableResource
}

func (c AccountConverter) createSecretCertificateNoPassVariable(resourceName string, description string) terraform.TerraformVariable {
	secretVariableResource := terraform.TerraformVariable{
		Name:        resourceName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: description,
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummyCertificateNoPass()
	}

	return secretVariableResource
}

func (c AccountConverter) createSecretCertificateB64Variable(resourceName string, description string, accountName string, dependencies *data.ResourceDetailsCollection) terraform.TerraformVariable {
	secretVariableResource := terraform.TerraformVariable{
		Name:        resourceName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: description,
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummyCertificateBase64()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: resourceName,
			ResourceName: accountName,
			ResourceType: c.GetResourceType(),
		})
	}

	return secretVariableResource
}

// writeData appends the data block for stateless modules
func (c AccountConverter) writeData(file *hclwrite.File, account octopus.Account, resourceName string) {
	terraformResource := c.buildData(resourceName, account)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c AccountConverter) getCount(stateless bool, resourceName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? 0 : 1}")
	}
	return nil
}

func (c AccountConverter) getAwsLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_aws_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_aws_account." + resourceName + ".id}"

}

func (c AccountConverter) getAwsDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_aws_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeAwsAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getAwsLookup(stateless, resourceName)
	resource.Dependency = c.getAwsDependency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " secret access key",
			Description:   "The AWS secret key associated with the account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "SecretAccessKey"),
			ParameterType: "SecretAccessKey",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformAwsAccount{
			Type:                            "octopusdeploy_aws_account",
			Name:                            resourceName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			ResourceName:                    account.Name,
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			AccessKey:                       account.AccessKey,
			SecretKey:                       &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The AWS secret key associated with the account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"secret_key"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getAzureServicePrincipalLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_azure_service_principal." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
}

func (c AccountConverter) getAzureServicePrincipalsDependency(stateless bool, resourceName string) string {
	if stateless {

		return "${octopusdeploy_azure_service_principal." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeAzureServicePrincipalAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getAzureServicePrincipalLookup(stateless, resourceName)
	resource.Dependency = c.getAzureServicePrincipalsDependency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " secret",
			Description:   "The Azure secret associated with the account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "Secret"),
			ParameterType: "Secret",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformAzureServicePrincipal{
			Type:                            "octopusdeploy_azure_service_principal",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			ApplicationId:                   account.ClientId,
			Password:                        &secretVariable,
			SubscriptionId:                  account.SubscriptionNumber,
			TenantId:                        account.TenantId,
			AzureEnvironment:                strutil.NilIfEmptyPointer(account.AzureEnvironment),
			ResourceManagerEndpoint:         strutil.NilIfEmptyPointer(account.ResourceManagementEndpointBaseUri),
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The Azure secret associated with the account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"password"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getAzureSubscriptionLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_azure_subscription_account." + resourceName + "[0].id}"

	}
	return "${octopusdeploy_azure_subscription_account." + resourceName + ".id}"
}

func (c AccountConverter) getAzureSubscriptionDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_azure_subscription_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeAzureSubscriptionAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getAzureSubscriptionLookup(stateless, resourceName)
	resource.Dependency = c.getAzureSubscriptionDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		certVariable := "${var." + resourceName + "_cert}"
		terraformResource := terraform.TerraformAzureSubscription{
			Type:                            "octopusdeploy_azure_subscription_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			ManagementEndpoint:              strutil.DefaultPointerIfEmptyOrNil(account.ServiceManagementEndpointBaseUri, "placeholder"),
			StorageEndpointSuffix:           strutil.DefaultPointerIfEmptyOrNil(account.ServiceManagementEndpointSuffix, "placeholder"),
			SubscriptionId:                  account.SubscriptionNumber,
			AzureEnvironment:                strutil.DefaultPointerIfEmptyOrNil(account.AzureEnvironment, "AzureCloud"),
			Certificate:                     &certVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretCertificateNoPassVariable(resourceName+"_cert", "The Azure certificate associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"certificate"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getGoogleCloudLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_gcp_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_gcp_account." + resourceName + ".id}"
}

func (c AccountConverter) getGoogleCloudDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_gcp_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeGoogleCloudAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getGoogleCloudLookup(stateless, resourceName)
	resource.Dependency = c.getGoogleCloudDependency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " JSON key",
			Description:   "The GCP JSON key associated with the account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "JsonKey"),
			ParameterType: "JsonKey",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformGcpAccount{
			Type:                            "octopusdeploy_gcp_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			JsonKey:                         &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The GCP JSON key associated with the account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"json_key"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getTokenLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_token_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_token_account." + resourceName + ".id}"
}

func (c AccountConverter) getTokenDpendency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_token_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeTokenAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getTokenLookup(stateless, resourceName)
	resource.Dependency = c.getTokenDpendency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " token",
			Description:   "The token associated with the account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "Token"),
			ParameterType: "Token",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformTokenAccount{
			Type:                            "octopusdeploy_token_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			Token:                           &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The token associated with the account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"token"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getUsernamePasswordLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_username_password_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_username_password_account." + resourceName + ".id}"

}

func (c AccountConverter) getUsernamePasswordDpendency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_username_password_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeUsernamePasswordAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getUsernamePasswordLookup(stateless, resourceName)
	resource.Dependency = c.getUsernamePasswordDpendency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " password",
			Description:   "The password associated with the account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "Password"),
			ParameterType: "Password",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformUsernamePasswordAccount{
			Type:                            "octopusdeploy_username_password_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			Username:                        account.Username,
			Password:                        &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The password associated with the account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"password"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getSshLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_ssh_key_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_ssh_key_account." + resourceName + ".id}"
}

func (c AccountConverter) getSshDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_ssh_key_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeSshAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getSshLookup(stateless, resourceName)
	resource.Dependency = c.getSshDependency(stateless, resourceName)
	resource.Parameters = []data.ResourceParameter{
		{
			VariableName:  resourceName,
			Label:         "Account " + account.Name + " certificate password",
			Description:   "The password associated with the certificate for account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "CertPassword"),
			ParameterType: "CertPassword",
			Sensitive:     true,
		},
		{
			VariableName:  resourceName + "_cert",
			Label:         "Account " + account.Name + " certificate data",
			Description:   "The certificate file for account " + account.Name,
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, account.Name, "CertData"),
			ParameterType: "CertData",
			Sensitive:     true,
		},
	}
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		certFileVariable := "${var." + resourceName + "_cert}"
		terraformResource := terraform.TerraformSshAccount{
			Type:                            "octopusdeploy_ssh_key_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			PrivateKeyFile:                  &certFileVariable,
			Username:                        account.Username,
			PrivateKeyPassphrase:            &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		// Because of https://github.com/OctopusDeployLabs/terraform-provider-octopusdeploy/issues/343
		secretVariableResource := c.createSecretCertificateB64Variable(resourceName, "The password associated with the certificate for account "+account.Name, account.Name, dependencies)

		certFileVariableResource := c.createSecretCertificateB64Variable(resourceName+"_cert", "The certificate file for account "+account.Name, account.Name, dependencies)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{"private_key_passphrase", "private_key_file"})

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(certFileVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) getAwsOidcLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_aws_openid_connect_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_aws_openid_connect_account." + resourceName + ".id}"
}

func (c AccountConverter) getAwsOidcDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_aws_openid_connect_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeAwsOidcAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) error {

	duration, err := intutil.ParseIntPointer(account.SessionDuration)

	if err != nil {
		return err
	}

	resource.Lookup = c.getAwsOidcLookup(stateless, resourceName)
	resource.Dependency = c.getAwsOidcDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformAwsOidcAccount{
			Type:                            "octopusdeploy_aws_openid_connect_account",
			Name:                            resourceName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			Count:                           c.getCount(stateless, resourceName),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			ResourceName:                    account.Name,
			Description:                     strutil.TrimPointer(account.Description),
			RoleArn:                         strutil.EmptyIfNil(account.RoleArn),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			ExecutionSubjectKeys:            account.DeploymentSubjectKeys,
			AccountTestSubjectKeys:          account.AccountTestSubjectKeys,
			HealthSubjectKeys:               account.HealthCheckSubjectKeys,
			SessionDuration:                 duration,
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{})

		file.Body().AppendBlock(accountBlock)

		return string(file.Bytes()), nil
	}

	return nil
}

func (c AccountConverter) getAzureOidcLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_azure_openid_connect." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_azure_openid_connect." + resourceName + ".id}"
}

func (c AccountConverter) getAzureOidcDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_azure_openid_connect." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeAzureOidcAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getAzureOidcLookup(stateless, resourceName)
	resource.Dependency = c.getAzureOidcDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformAzureOidcSubscription{
			Type:                            "octopusdeploy_azure_openid_connect",
			Name:                            resourceName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			Count:                           c.getCount(stateless, resourceName),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			ResourceName:                    account.Name,
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			SubscriptionId:                  account.SubscriptionNumber,
			AzureEnvironment:                account.AzureEnvironment,
			Audience:                        account.Audience,
			AccountTestSubjectKeys:          account.AccountTestSubjectKeys,
			ExecutionSubjectKeys:            account.DeploymentSubjectKeys,
			HealthSubjectKeys:               account.HealthCheckSubjectKeys,
			AuthenticationEndpoint:          strutil.NilIfEmptyPointer(account.ActiveDirectoryEndpointBaseUri),
			ResourceManagerEndpoint:         strutil.NilIfEmptyPointer(account.ResourceManagementEndpointBaseUri),
			TenantId:                        strutil.EmptyIfNil(account.TenantId),
			ApplicationId:                   strutil.EmptyIfNil(account.ClientId),
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{})

		file.Body().AppendBlock(accountBlock)

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) writeLifecycleAttributes(accountBlock *hclwrite.Block, stateless bool, dummyVars []string) {
	if c.DummySecretVariableValues || stateless {

		ignoreAll := terraform.EmptyBlock{}
		lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
		accountBlock.Body().AppendBlock(lifecycleBlock)

		if c.DummySecretVariableValues {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "["+strings.Join(dummyVars, ", ")+"]")
		}

		if stateless {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
		}
	}
}

func (c AccountConverter) getGenericOidcLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data.octopusdeploy_accounts." + resourceName + ".accounts) != 0 ? data.octopusdeploy_accounts." + resourceName + ".accounts[0].id : octopusdeploy_generic_oidc_account." + resourceName + "[0].id}"
	}
	return "${octopusdeploy_generic_oidc_account." + resourceName + ".id}"
}

func (c AccountConverter) getGenericOidcDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${octopusdeploy_generic_oidc_account." + resourceName + "}"
	}

	return ""
}

func (c AccountConverter) writeGenericOidcAccount(stateless bool, resource *data.ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *data.ResourceDetailsCollection) {

	resource.Lookup = c.getAwsOidcLookup(stateless, resourceName)
	resource.Dependency = c.getAwsOidcDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformGenericOicdAccount{
			Type:                            "octopusdeploy_aws_openid_connect_account",
			Name:                            resourceName,
			Id:                              strutil.InputPointerIfEnabled(c.IncludeIds, &account.Id),
			Count:                           c.getCount(stateless, resourceName),
			SpaceId:                         strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", account.SpaceId)),
			ResourceName:                    account.Name,
			Description:                     strutil.TrimPointer(account.Description),
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			ExecutionSubjectKeys:            account.DeploymentSubjectKeys,
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
		}

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		c.writeLifecycleAttributes(accountBlock, stateless, []string{})

		file.Body().AppendBlock(accountBlock)

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) exportDependencies(target octopus.Account, dependencies *data.ResourceDetailsCollection) error {

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err := c.EnvironmentConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the tenants
	for _, e := range target.TenantIds {
		err := c.TenantConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
