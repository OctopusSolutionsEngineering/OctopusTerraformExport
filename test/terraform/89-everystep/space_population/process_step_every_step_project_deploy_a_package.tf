variable "project_every_step_project_step_deploy_a_package_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy a Package in project Every Step Project"
  default     = "mypackage"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_a_package" {
  name                  = "Deploy a Package"
  type                  = "Octopus.TentaclePackage"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step copies a package to server or virtual machine."
  package_requirement   = "LetOctopusDecide"
  primary_package       = {
    acquisition_location = "Server",
    feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}",
    id = null,
    package_id = "${var.project_every_step_project_step_deploy_a_package_packageid}",
    properties = {
      SelectionMode = "immediate"
    }
  }
  slug                  = "deploy-a-package"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "windows-server"
      }
  execution_properties  = {
        "Octopus.Action.Package.AutomaticallyRunConfigurationTransformationFiles" = "True"
        "Octopus.Action.Package.AutomaticallyUpdateAppSettingsAndConnectionStrings" = "True"
        "Octopus.Action.EnabledFeatures" = ",Octopus.Features.ConfigurationTransforms,Octopus.Features.ConfigurationVariables"
      }
}
