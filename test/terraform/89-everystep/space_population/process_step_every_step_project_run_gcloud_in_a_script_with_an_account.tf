resource "octopusdeploy_process_step" "process_step_every_step_project_run_gcloud_in_a_script_with_an_account" {
  name                  = "Run gcloud in a Script with an account"
  type                  = "Octopus.GoogleCloudScripting"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This is a script that runs against Google Cloud Platform (GCP) resources using an account."
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-gcloud-in-a-script-with-an-account"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "Bash"
        "Octopus.Action.GoogleCloud.Region" = "australia-southeast1"
        "Octopus.Action.GoogleCloud.Zone" = "australia-southeast1-a"
        "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "False"
        "Octopus.Action.GoogleCloud.Project" = "ProjectID"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.GoogleCloudAccount.Variable" = "Example.GCP.Variable"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptBody" = "echo \"Hi\""
      }
}
