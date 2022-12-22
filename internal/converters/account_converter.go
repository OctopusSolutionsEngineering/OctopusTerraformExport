package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AccountConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c AccountConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Account]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	accountMap := map[string]string{}

	for _, account := range collection.Items {
		resourceName := "account_" + util.SanitizeName(account.Slug)

		// Assume the default lifecycle already exists
		if account.AccountType == "AmazonWebServicesAccount" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformAwsAccount{
				Type:                            "octopusdeploy_aws_account",
				Name:                            resourceName,
				SpaceId:                         c.SpaceResourceName,
				Description:                     account.Description,
				Environments:                    nil,
				TenantTags:                      account.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
				AccessKey:                       account.AccessKey,
				SecretKey:                       &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The AWS secret key associated with the account " + account.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(secretVariableResource, "variable"))

			results[resourceName+".tf"] = string(file.Bytes())
			accountMap[account.Id] = "${octopusdeploy_aws_account." + resourceName + ".id}"
		}

		if account.AccountType == "AzureServicePrincipal" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformAzureServicePrincipal{
				Type:                            "octopusdeploy_azure_service_principal",
				Name:                            resourceName,
				SpaceId:                         c.SpaceResourceName,
				Description:                     account.Description,
				Environments:                    nil,
				TenantTags:                      account.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
				ApplicationId:                   account.ClientId,
				Password:                        &secretVariable,
				SubscriptionId:                  account.SubscriptionNumber,
				TenantId:                        account.TenantId,
				AzureEnvironment:                account.AzureEnvironment,
				ResourceManagerEndpoint:         account.ResourceManagementEndpointBaseUri,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The Azure secret associated with the account " + account.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(secretVariableResource, "variable"))

			results[resourceName+".tf"] = string(file.Bytes())
			accountMap[account.Id] = "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
		}

		if account.AccountType == "GoogleCloudAccount" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformGcpAccount{
				Type:                            "octopusdeploy_gcp_account",
				Name:                            resourceName,
				SpaceId:                         c.SpaceResourceName,
				Description:                     account.Description,
				Environments:                    nil,
				TenantTags:                      account.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
				JsonKey:                         &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The GCP JSON key associated with the account " + account.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(secretVariableResource, "variable"))

			results[resourceName+".tf"] = string(file.Bytes())
			accountMap[account.Id] = "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
		}

		if account.AccountType == "Token" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformTokenAccount{
				Type:                            "octopusdeploy_token_account",
				Name:                            resourceName,
				SpaceId:                         c.SpaceResourceName,
				Description:                     account.Description,
				Environments:                    nil,
				TenantTags:                      account.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
				Token:                           &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The token associated with the account " + account.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(secretVariableResource, "variable"))

			results[resourceName+".tf"] = string(file.Bytes())
			accountMap[account.Id] = "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
		}

		if account.AccountType == "UsernamePassword" {
			secretVariable := "${var." + resourceName + "}"
			terraformResource := terraform.TerraformUsernamePasswordAccount{
				Type:                            "octopusdeploy_username_password_account",
				Name:                            resourceName,
				SpaceId:                         c.SpaceResourceName,
				Description:                     account.Description,
				Environments:                    nil,
				TenantTags:                      account.TenantTags,
				Tenants:                         nil,
				TenantedDeploymentParticipation: account.TenantedDeploymentParticipation,
				Username:                        account.Username,
				Password:                        &secretVariable,
			}

			secretVariableResource := terraform.TerraformVariable{
				Name:        resourceName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The password associated with the account " + account.Name,
			}

			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
			file.Body().AppendBlock(gohcl.EncodeAsBlock(secretVariableResource, "variable"))

			results[resourceName+".tf"] = string(file.Bytes())
			accountMap[account.Id] = "${octopusdeploy_azure_service_principal." + resourceName + ".id}"
		}

		//todo: octopusdeploy_ssh_key_account, octopusdeploy_azure_subscription_account
	}

	return results, accountMap, nil
}

func (c AccountConverter) GetResourceType() string {
	return "Accounts"
}
