package converters

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
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

func (c AccountConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Account]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Account: " + resource.Id)
		err = c.toHcl(resource, false, dependencies)

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

	resource := octopus2.Account{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Account: " + resource.Id)
	return c.toHcl(resource, true, dependencies)
}

func (c AccountConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Account{}
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
		terraformResource := terraform2.TerraformAccountData{
			Type:        "octopusdeploy_accounts",
			Name:        resourceName,
			Ids:         nil,
			PartialName: resource.Name,
			Skip:        0,
			Take:        1,
		}

		// Google account types are not defined in the data resource (this is a bug), so don't use it
		if resource.AccountType != "GoogleCloudAccount" {
			terraformResource.AccountType = resource.AccountType
		}

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an account called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.accounts) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) toHcl(resource octopus2.Account, recursive bool, dependencies *ResourceDetailsCollection) error {
	if recursive {
		c.exportDependencies(resource, dependencies)
	}

	resourceName := "account_" + sanitizer.SanitizeName(resource.Name)

	thisResource := ResourceDetails{}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	if resource.AccountType == "AmazonWebServicesAccount" {
		thisResource.Lookup = "${octopusdeploy_aws_account." + resourceName + ".id}"
	} else if resource.AccountType == "AzureServicePrincipal" {
		thisResource.Lookup = "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
	} else if resource.AccountType == "AzureSubscription" {
		thisResource.Lookup = "${octopusdeploy_azure_subscription_account." + resourceName + ".id}"
	} else if resource.AccountType == "GoogleCloudAccount" {
		thisResource.Lookup = "${octopusdeploy_gcp_account." + resourceName + ".id}"
	} else if resource.AccountType == "Token" {
		thisResource.Lookup = "${octopusdeploy_token_account." + resourceName + ".id}"
	} else if resource.AccountType == "UsernamePassword" {
		thisResource.Lookup = "${octopusdeploy_username_password_account." + resourceName + ".id}"
	} else if resource.AccountType == "SshKeyPair" {
		thisResource.Lookup = "${octopusdeploy_ssh_key_account." + resourceName + ".id}"
	}
	thisResource.ToHcl = func() (string, error) {

		// Assume the default lifecycle already exists
		if resource.AccountType == "AmazonWebServicesAccount" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform2.TerraformAwsAccount{
				Type:                            "octopusdeploy_aws_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				AccessKey:                       resource.AccessKey,
				SecretKey:                       &secretVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The AWS secret key associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_aws_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[secret_key]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "AzureServicePrincipal" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform2.TerraformAzureServicePrincipal{
				Type:                            "octopusdeploy_azure_service_principal",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				ApplicationId:                   resource.ClientId,
				Password:                        &secretVariable,
				SubscriptionId:                  resource.SubscriptionNumber,
				TenantId:                        resource.TenantId,
				AzureEnvironment:                strutil.NilIfEmptyPointer(resource.AzureEnvironment),
				ResourceManagerEndpoint:         strutil.NilIfEmptyPointer(resource.ResourceManagementEndpointBaseUri),
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The Azure secret associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_azure_service_principal." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[password]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "AzureSubscription" {
			certVariable := "${var." + resourceName + "_cert}"
			terraformResource := terraform2.TerraformAzureSubscription{
				Type:                            "octopusdeploy_azure_subscription_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				ManagementEndpoint:              strutil.EmptyIfNil(resource.ServiceManagementEndpointBaseUri),
				StorageEndpointSuffix:           strutil.EmptyIfNil(resource.ServiceManagementEndpointSuffix),
				SubscriptionId:                  resource.SubscriptionNumber,
				AzureEnvironment:                strutil.NilIfEmptyPointer(resource.AzureEnvironment),
				Certificate:                     &certVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName + "_cert",
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The Azure certificate associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_azure_subscription_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[certificate]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "GoogleCloudAccount" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform2.TerraformGcpAccount{
				Type:                            "octopusdeploy_gcp_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				JsonKey:                         &secretVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The GCP JSON key associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_gcp_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[json_key]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "Token" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform2.TerraformTokenAccount{
				Type:                            "octopusdeploy_token_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				Token:                           &secretVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The token associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_token_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[token]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "UsernamePassword" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform2.TerraformUsernamePasswordAccount{
				Type:                            "octopusdeploy_username_password_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				Username:                        resource.Username,
				Password:                        &secretVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password associated with the account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_username_password_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[password]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "SshKeyPair" {
			secretVariable := "${var." + resourceName + "}"
			certFileVariable := "${var." + resourceName + "_cert}"
			terraformResource := terraform2.TerraformSshAccount{
				Type:                            "octopusdeploy_ssh_key_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    dependencies.GetResources("Environments", resource.EnvironmentIds...),
				TenantTags:                      c.Excluder.FilteredTenantTags(resource.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				Tenants:                         dependencies.GetResources("Tenants", resource.TenantIds...),
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				PrivateKeyFile:                  &certFileVariable,
				Username:                        resource.Username,
				PrivateKeyPassphrase:            &secretVariable,
			}

			secretVariableResource := terraform2.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password associated with the certificate for account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			certFileVariableResource := terraform2.TerraformVariable{
				Name:        resourceName + "_cert",
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The certificate file for account " + resource.Name,
			}

			if c.DummySecretVariableValues {
				secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
			}

			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_ssh_key_account." + resourceName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			accountBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, accountBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				hcl.WriteLifecycleAttribute(accountBlock, "[private_key_passphrase, private_key_file]")
			}

			file.Body().AppendBlock(accountBlock)

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			certFileVariableResourceBlock := gohcl.EncodeAsBlock(certFileVariableResource, "variable")
			hcl.WriteUnquotedAttribute(certFileVariableResourceBlock, "type", "string")
			file.Body().AppendBlock(certFileVariableResourceBlock)

			return string(file.Bytes()), nil
		}

		return "", errors.New("found unsupported account type: " + resource.AccountType)
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) GetResourceType() string {
	return "Accounts"
}

func (c AccountConverter) exportDependencies(target octopus2.Account, dependencies *ResourceDetailsCollection) error {

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
