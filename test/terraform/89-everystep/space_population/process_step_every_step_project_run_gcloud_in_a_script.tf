resource "octopusdeploy_process_step" "process_step_every_step_project_run_gcloud_in_a_script" {
  name                  = "Run gcloud in a Script"
  type                  = "Octopus.GoogleCloudScripting"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This is a script that runs against Google Cloud Platform (GCP) resources"
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-gcloud-in-a-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.GoogleCloud.Zone" = "australia-southeast1-a"
        "Octopus.Action.GoogleCloud.Project" = "ProjectID"
        "Octopus.Action.Script.Syntax" = "Bash"
        "Octopus.Action.Script.ScriptBody" = "echo \"Hi\""
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.GoogleCloud.Region" = "australia-southeast1"
        "Octopus.Action.RunOnServer" = "true"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "True"
        "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
      }
}
