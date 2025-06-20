variable "project_every_step_project_step_deploy_to_iis_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy to IIS in project Every Step Project"
  default     = "webapp"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_to_iis" {
  name                  = "Deploy to IIS"
  type                  = "Octopus.IIS"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys a Windows IIS application."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_deploy_to_iis_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-to-iis"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "windows-server"
      }
  execution_properties  = {
        "Octopus.Action.IISWebSite.WebApplication.ApplicationPoolFrameworkVersion" = "v4.0"
        "Octopus.Action.IISWebSite.WebRootType" = "packageRoot"
        "Octopus.Action.IISWebSite.EnableBasicAuthentication" = "False"
        "Octopus.Action.Package.AutomaticallyRunConfigurationTransformationFiles" = "True"
        "Octopus.Action.IISWebSite.EnableWindowsAuthentication" = "True"
        "Octopus.Action.IISWebSite.ApplicationPoolFrameworkVersion" = "v4.0"
        "Octopus.Action.EnabledFeatures" = ",Octopus.Features.IISWebSite,Octopus.Features.ConfigurationTransforms,Octopus.Features.ConfigurationVariables"
        "Octopus.Action.Package.AutomaticallyUpdateAppSettingsAndConnectionStrings" = "True"
        "Octopus.Action.IISWebSite.EnableAnonymousAuthentication" = "False"
        "Octopus.Action.IISWebSite.WebApplication.ApplicationPoolIdentityType" = "ApplicationPoolIdentity"
        "Octopus.Action.IISWebSite.ApplicationPoolIdentityType" = "ApplicationPoolIdentity"
        "Octopus.Action.IISWebSite.CreateOrUpdateWebSite" = "True"
        "Octopus.Action.IISWebSite.StartApplicationPool" = "True"
        "Octopus.Action.IISWebSite.ApplicationPoolName" = "apppool"
        "Octopus.Action.IISWebSite.DeploymentType" = "webSite"
        "Octopus.Action.IISWebSite.WebSiteName" = "webapp"
        "Octopus.Action.IISWebSite.Bindings" = jsonencode([
        {
        "host" = ""
        "thumbprint" = null
        "certificateVariable" = null
        "requireSni" = "False"
        "enabled" = "True"
        "protocol" = "http"
        "port" = "80"
                },
        ])
        "Octopus.Action.IISWebSite.StartWebSite" = "True"
      }
}
