variable "project_every_step_project_step_run_a_script_from_a_package_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Run a Script from a package in project Every Step Project"
  default     = "scripts"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_run_a_script_from_a_package" {
  name                  = "Run a Script from a package"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "An example step that is configured to run a script from a package. Note how this step defines a primary_package."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_run_a_script_from_a_package_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "run-a-script-from-a-package-1"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = ["Tag Set/tag", "Tag Set/tag2"]
  worker_pool_id        = "${data.octopusdeploy_worker_pools.workerpool_hosted_windows.worker_pools[0].id}"
  properties            = {
      }
  execution_properties  = {
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Package"
        "Octopus.Action.Script.ScriptFileName" = "MyScript.ps1"
      }
}
