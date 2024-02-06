package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type AccountConverter struct {
	Client                    client.OctopusClient
	EnvironmentConverter      ConverterById
	TenantConverter           ConverterById
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           TagSetConverter
}

func (c AccountConverter) AllToHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c AccountConverter) AllToStatelessHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c AccountConverter) allToHcl(stateless bool, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Account]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Account: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AccountConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Account{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Account: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c AccountConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Account{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	resourceName := "account_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
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

// toHcl adds this resource to the list of dependencies.
// account is the Octopus account object to be serialized
// recursive indicates if any transient dependencies are to be serialized
// stateless indicates if the resource is to be exported for use with a stateless Terraform transaction (i.e. where the
// Terraform state is not maintained between apply commands)
// dependencies maintains the collection of exported Terraform resources
func (c AccountConverter) toHcl(account octopus.Account, recursive bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	if recursive {
		err := c.exportDependencies(account, dependencies)

		if err != nil {
			return err
		}
	}

	resourceName := "account_" + sanitizer.SanitizeName(account.Name)

	thisResource := ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = account.Id
	thisResource.ResourceType = c.GetResourceType()

	if account.AccountType == "AmazonWebServicesAccount" {
		c.writeAwsAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "AzureServicePrincipal" {
		c.writeAzureServicePrincipalAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "AzureSubscription" {
		c.writeAzureSubscriptionAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "GoogleCloudAccount" {
		c.writeGoogleCloudAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "Token" {
		c.writeTokenAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "UsernamePassword" {
		c.writeUsernamePasswordAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	} else if account.AccountType == "SshKeyPair" {
		c.writeSshAccount(stateless, &thisResource, resourceName, account, recursive, dependencies)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) GetResourceType() string {
	return "Accounts"
}

func (c AccountConverter) createSecretVariable(resourceName string, description string) terraform.TerraformVariable {
	secretVariableResource := terraform.TerraformVariable{
		Name:        resourceName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: description,
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
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

func (c AccountConverter) createSecretCertificateB64Variable(resourceName string, description string) terraform.TerraformVariable {
	secretVariableResource := terraform.TerraformVariable{
		Name:        resourceName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: description,
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummyCertificateBase64()
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

func (c AccountConverter) writeAwsAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getAwsLookup(stateless, resourceName)
	resource.Dependency = c.getAwsDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformAwsAccount{
			Type:                            "octopusdeploy_aws_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			AccessKey:                       account.AccessKey,
			SecretKey:                       &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The AWS secret key associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_aws_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[secret_key]")
		}

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

func (c AccountConverter) writeAzureServicePrincipalAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getAzureServicePrincipalLookup(stateless, resourceName)
	resource.Dependency = c.getAzureServicePrincipalsDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformAzureServicePrincipal{
			Type:                            "octopusdeploy_azure_service_principal",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
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

		secretVariableResource := c.createSecretVariable(resourceName, "The Azure secret associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_azure_service_principal", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[password]")
		}

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

func (c AccountConverter) writeAzureSubscriptionAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getAzureSubscriptionLookup(stateless, resourceName)
	resource.Dependency = c.getAzureSubscriptionDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		certVariable := "${var." + resourceName + "_cert}"
		terraformResource := terraform.TerraformAzureSubscription{
			Type:                            "octopusdeploy_azure_subscription_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			ManagementEndpoint:              strutil.EmptyIfNil(account.ServiceManagementEndpointBaseUri),
			StorageEndpointSuffix:           strutil.EmptyIfNil(account.ServiceManagementEndpointSuffix),
			SubscriptionId:                  account.SubscriptionNumber,
			// A value is required, and an empty upstream string means "AzureCloud"
			AzureEnvironment: strutil.DefaultIfEmptyOrNil(account.AzureEnvironment, "AzureCloud"),
			Certificate:      &certVariable,
			Count:            c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretCertificateNoPassVariable(resourceName+"_cert", "The Azure certificate associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_azure_subscription_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[certificate]")
		}

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

func (c AccountConverter) writeGoogleCloudAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getGoogleCloudLookup(stateless, resourceName)
	resource.Dependency = c.getGoogleCloudDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformGcpAccount{
			Type:                            "octopusdeploy_gcp_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			JsonKey:                         &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The GCP JSON key associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_gcp_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[json_key]")
		}

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

func (c AccountConverter) writeTokenAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getTokenLookup(stateless, resourceName)
	resource.Dependency = c.getTokenDpendency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformTokenAccount{
			Type:                            "octopusdeploy_token_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			Token:                           &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The token associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_token_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[token]")
		}

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

func (c AccountConverter) writeUsernamePasswordAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getUsernamePasswordLookup(stateless, resourceName)
	resource.Dependency = c.getUsernamePasswordDpendency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		terraformResource := terraform.TerraformUsernamePasswordAccount{
			Type:                            "octopusdeploy_username_password_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
			Environments:                    dependencies.GetResources("Environments", account.EnvironmentIds...),
			TenantTags:                      c.Excluder.FilteredTenantTags(account.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			Tenants:                         dependencies.GetResources("Tenants", account.TenantIds...),
			TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
			Username:                        account.Username,
			Password:                        &secretVariable,
			Count:                           c.getCount(stateless, resourceName),
		}

		secretVariableResource := c.createSecretVariable(resourceName, "The password associated with the account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_username_password_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[password]")
		}

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

func (c AccountConverter) writeSshAccount(stateless bool, resource *ResourceDetails, resourceName string, account octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) {

	resource.Lookup = c.getSshLookup(stateless, resourceName)
	resource.Dependency = c.getSshDependency(stateless, resourceName)
	resource.ToHcl = func() (string, error) {
		secretVariable := "${var." + resourceName + "}"
		certFileVariable := "${var." + resourceName + "_cert}"
		terraformResource := terraform.TerraformSshAccount{
			Type:                            "octopusdeploy_ssh_key_account",
			Name:                            resourceName,
			ResourceName:                    account.Name,
			Description:                     account.Description,
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
		secretVariableResource := c.createSecretCertificateB64Variable(resourceName, "The password associated with the certificate for account "+account.Name)

		certFileVariableResource := c.createSecretCertificateB64Variable(resourceName+"_cert", "The certificate file for account "+account.Name)

		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, account, resourceName)
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), "octopusdeploy_ssh_key_account", account.Name, resourceName))

		accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDeleteAttribute(accountBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}

		// When using dummy values, we expect the secrets will be updated later
		if c.DummySecretVariableValues {
			hcl.WriteLifecycleAttribute(accountBlock, "[private_key_passphrase, private_key_file]")
		}

		file.Body().AppendBlock(accountBlock)
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(secretVariableResource))
		file.Body().AppendBlock(hcl.EncodeTerraformVariable(certFileVariableResource))

		return string(file.Bytes()), nil
	}
}

func (c AccountConverter) exportDependencies(target octopus.Account, dependencies *ResourceDetailsCollection) error {

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
