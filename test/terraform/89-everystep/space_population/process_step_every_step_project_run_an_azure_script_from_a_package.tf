variable "project_every_step_project_step_run_an_azure_script_from_a_package_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Run an Azure Script from a package in project Every Step Project"
  default     = "AzureScripts"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_run_an_azure_script_from_a_package" {
  name                  = "Run an Azure Script from a package"
  type                  = "Octopus.AzurePowerShell"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This is an example step that runs a script against Azure resources from a package. Note how this step defines a primary_package. Note how this step defines the \"Octopus.Action.Azure.AccountId\" property."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_run_an_azure_script_from_a_package_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "run-an-azure-script-from-a-package"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = ["Business Units/Billing", "Business Units/Engineering", "Business Units/HR", "Business Units/Insurance", "Cities/London", "Cities/Madrid", "Cities/Sydney", "Cities/Washington", "Cities/Wellington", "Regions/ANZ", "Regions/Asia", "Regions/Europe", "Regions/US"]
  worker_pool_id        = "${data.octopusdeploy_worker_pools.workerpool_hosted_ubuntu.worker_pools[0].id}"
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Script.ScriptFileName" = "CreaeResourceGroup.ps1"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Script.ScriptSource" = "Package"
        "Octopus.Action.Azure.AccountId" = "${octopusdeploy_azure_openid_connect.account_azure.id}"
        "Octopus.Action.RunOnServer" = "true"
      }
}
