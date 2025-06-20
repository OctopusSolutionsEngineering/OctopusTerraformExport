resource "octopusdeploy_process_step" "process_step_every_step_project_run_an_azure_script" {
  name                  = "Run an Azure Script"
  type                  = "Octopus.AzurePowerShell"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This is an example step that runs an inline script against Azure resources. Note how this step does not defined a primary_package. Note how this step defines the \"Octopus.Action.Azure.AccountId\" property."
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-an-azure-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = ["Business Units/Billing", "Business Units/Engineering", "Business Units/HR", "Business Units/Insurance", "Cities/London", "Cities/Madrid", "Cities/Sydney", "Cities/Washington", "Cities/Wellington", "Regions/ANZ", "Regions/Asia", "Regions/Europe", "Regions/US"]
  worker_pool_id        = "${octopusdeploy_static_worker_pool.ubuntu.id}"
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Script.ScriptBody" = "# Note how dollar signs are escaped with another dollar sign\n$myvariable = \"hi\"\necho \"$myvariable\""
        "Octopus.Action.RunOnServer" = "true"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "Bash"
        "Octopus.Action.Azure.AccountId" = "${octopusdeploy_azure_openid_connect.account_azure.id}"
      }
}
