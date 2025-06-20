resource "octopusdeploy_process_step" "process_step_every_step_project_example_runbook_run_a_script" {
  name                  = "Run a Script"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_every_step_project_example_runbook.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-a-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_id        = "${data.octopusdeploy_worker_pools.workerpool_hosted_windows.worker_pools[0].id}"
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Script.Syntax" = "PowerShell"
        "Octopus.Action.Script.ScriptBody" = "echo \"This is an example script step\""
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Inline"
      }
}
