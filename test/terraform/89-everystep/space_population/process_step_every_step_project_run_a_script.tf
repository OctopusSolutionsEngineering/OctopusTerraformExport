resource "octopusdeploy_process_step" "process_step_every_step_project_run_a_script" {
  name                  = "Run a Script"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "An example step that runs an inline script. This step is configured to only run for tenants with the London tenant tag. Note how this step does not defined a primary_package."
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-a-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = ["Cities/London"]
  worker_pool_id        = "${data.octopusdeploy_worker_pools.workerpool_hosted_windows.worker_pools[0].id}"
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Script.Syntax" = "Bash"
        "Octopus.Action.Script.ScriptBody" = "echo \"Hello World!\"\n\nVARIABLE=\"test\"\n\n# Pay attention to how the $ character is escaped when defined in Terraform\necho \"$${VARIABLE}\"\n\n# Pay attention to how the percent character is escaped when defined in Terraform\ncurl -w \"%%{http_code}\" http://example.org"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Inline"
      }
}
