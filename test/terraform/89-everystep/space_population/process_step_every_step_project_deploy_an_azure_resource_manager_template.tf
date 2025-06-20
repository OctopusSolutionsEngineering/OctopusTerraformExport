resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_an_azure_resource_manager_template" {
  name                  = "Deploy an Azure Resource Manager template"
  type                  = "Octopus.AzureResourceGroup"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys an Azure ARM template"
  package_requirement   = "LetOctopusDecide"
  slug                  = "deploy-an-azure-resource-manager-template"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Azure.ResourceGroupDeploymentMode" = "Incremental"
        "Octopus.Action.Azure.TemplateSource" = "Inline"
        "Octopus.Action.Azure.ResourceGroupTemplateParameters" = jsonencode({        })
        "Octopus.Action.Azure.AccountId" = "${octopusdeploy_azure_openid_connect.account_azure.id}"
        "Octopus.Action.Azure.ResourceGroupName" = "my-resource-group"
        "Octopus.Action.Azure.ResourceGroupTemplate" = jsonencode({
        "$schema" = "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#"
        "contentVersion" = "1.0.0.0"
        "resources" = []
                })
        "OctopusUseBundledTooling" = "False"
      }
}
