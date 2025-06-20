resource "octopusdeploy_process_step" "process_step_every_step_project_run_a_script_on_failure" {
  name                  = "Run a Script on Failure"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Failure"
  environments          = null
  excluded_environments = null
  notes                 = "This is an example of a step that run when the previous step fails."
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-a-script-on-failure"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_id        = "${octopusdeploy_static_worker_pool.windows.id}"
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "PowerShell"
        "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.RunOnServer" = "true"
      }
}
