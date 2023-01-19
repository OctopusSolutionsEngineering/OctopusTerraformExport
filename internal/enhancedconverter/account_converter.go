package enhancedconverter

import (
	"errors"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AccountConverter struct {
	Client client.OctopusClient
}

func (c AccountConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Account]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AccountConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.Account{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c AccountConverter) toHcl(resource octopus.Account, recursive bool, dependencies *ResourceDetailsCollection) error {
	// TODO: export environments

	resourceName := "account_" + util.SanitizeName(resource.Name)

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
			terraformResource := terraform.TerraformAwsAccount{
				Type:                            "octopusdeploy_aws_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				AccessKey:                       resource.AccessKey,
				SecretKey:                       &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The AWS secret key associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "AzureServicePrincipal" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformAzureServicePrincipal{
				Type:                            "octopusdeploy_azure_service_principal",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				ApplicationId:                   resource.ClientId,
				Password:                        &secretVariable,
				SubscriptionId:                  resource.SubscriptionNumber,
				TenantId:                        resource.TenantId,
				AzureEnvironment:                util.NilIfEmptyPointer(resource.AzureEnvironment),
				ResourceManagerEndpoint:         util.NilIfEmptyPointer(resource.ResourceManagementEndpointBaseUri),
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The Azure secret associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "AzureSubscription" {
			certVariable := "${var." + resourceName + "_cert}"
			terraformResource := terraform.TerraformAzureSubscription{
				Type:                            "octopusdeploy_azure_subscription_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				ManagementEndpoint:              util.EmptyIfNil(resource.ResourceManagementEndpointBaseUri),
				StorageEndpointSuffix:           util.EmptyIfNil(resource.ServiceManagementEndpointSuffix),
				SubscriptionId:                  resource.SubscriptionNumber,
				AzureEnvironment:                util.NilIfEmptyPointer(resource.AzureEnvironment),
				Certificate:                     &certVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName + "_cert",
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The Azure certificate associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "GoogleCloudAccount" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformGcpAccount{
				Type:                            "octopusdeploy_gcp_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				JsonKey:                         &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The GCP JSON key associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "Token" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformTokenAccount{
				Type:                            "octopusdeploy_token_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				Token:                           &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The token associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "UsernamePassword" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformUsernamePasswordAccount{
				Type:                            "octopusdeploy_username_password_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				Username:                        resource.Username,
				Password:                        &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password associated with the account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		if resource.AccountType == "SshKeyPair" {
			secretVariable := "${var." + resourceName + "}"
			certFileVariable := "${var." + resourceName + "_cert}"
			terraformResource := terraform.TerraformSshAccount{
				Type:                            "octopusdeploy_ssh_key_account",
				Name:                            resourceName,
				ResourceName:                    resource.Name,
				Description:                     resource.Description,
				Environments:                    nil,
				TenantTags:                      resource.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: resource.TenantedDeploymentParticipation,
				PrivateKeyFile:                  &certFileVariable,
				Username:                        resource.Username,
				PrivateKeyPassphrase:            &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password associated with the certificate for account " + resource.Name,
			}

			certFileVariableResource := terraform.TerraformVariable{
				Name:        resourceName + "_cert",
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The certificate file for account " + resource.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			certFileVariableResourceBlock := gohcl.EncodeAsBlock(certFileVariableResource, "variable")
			util.WriteUnquotedAttribute(certFileVariableResourceBlock, "type", "string")
			file.Body().AppendBlock(certFileVariableResourceBlock)

			return string(file.Bytes()), nil
		}

		return "", errors.New("found unsupported account type")
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AccountConverter) GetResourceType() string {
	return "Accounts"
}
