variable "project_every_step_project_step_deploy_a_windows_service_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy a Windows Service in project Every Step Project"
  default     = "myservice"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_a_windows_service" {
  name                  = "Deploy a Windows Service"
  type                  = "Octopus.WindowsService"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys a windows service."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_deploy_a_windows_service_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-a-windows-service"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "windows-server"
      }
  execution_properties  = {
        "Octopus.Action.Package.AutomaticallyRunConfigurationTransformationFiles" = "True"
        "Octopus.Action.WindowsService.CreateOrUpdateService" = "True"
        "Octopus.Action.WindowsService.DisplayName" = "My sample Windows service"
        "Octopus.Action.WindowsService.ServiceAccount" = "LocalSystem"
        "Octopus.Action.WindowsService.Description" = "This is a sample deployment of a Windows service"
        "Octopus.Action.WindowsService.ServiceName" = "My Service"
        "Octopus.Action.WindowsService.StartMode" = "auto"
        "Octopus.Action.Package.AutomaticallyUpdateAppSettingsAndConnectionStrings" = "True"
        "Octopus.Action.EnabledFeatures" = ",Octopus.Features.WindowsService,Octopus.Features.ConfigurationTransforms,Octopus.Features.ConfigurationVariables"
        "Octopus.Action.WindowsService.ExecutablePath" = "myapp.exe"
        "Octopus.Action.WindowsService.DesiredStatus" = "Default"
      }
}
