variable "project_every_step_project_step_deploy_an_azure_web_app__web_deploy__packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy an Azure Web App (Web Deploy) in project Every Step Project"
  default     = "MyAzureWebApp"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_an_azure_web_app__web_deploy_" {
  name                  = "Deploy an Azure Web App (Web Deploy)"
  type                  = "Octopus.AzureWebApp"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys an application an an Azure Web App. Note how this step does not define any scripts. Note how this step defined the \"target_roles\" attribute."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_deploy_an_azure_web_app__web_deploy__packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-an-azure-web-app-web-deploy"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "AzureWebApp"
      }
  execution_properties  = {
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Azure.UseChecksum" = "False"
        "Octopus.Action.RunOnServer" = "true"
      }
}
