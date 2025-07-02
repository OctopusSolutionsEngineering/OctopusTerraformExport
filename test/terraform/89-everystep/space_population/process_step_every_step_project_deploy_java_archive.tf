variable "project_every_step_project_step_deploy_java_archive_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy Java Archive in project Every Step Project"
  default     = "MyJavaApp"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_java_archive" {
  name                  = "Deploy Java Archive"
  type                  = "Octopus.JavaArchive"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys a Java WAR or JAR file to the filesystem."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_deploy_java_archive_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-java-archive"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "JavaAppServer"
      }
  execution_properties  = {
        "Octopus.Action.JavaArchive.DeployExploded" = "False"
        "Octopus.Action.Package.JavaArchiveCompression" = "True"
        "Octopus.Action.EnabledFeatures" = ",Octopus.Features.SubstituteInFiles"
        "Octopus.Action.Package.UseCustomInstallationDirectory" = "False"
        "Octopus.Action.Package.CustomInstallationDirectoryShouldBePurgedBeforeDeployment" = "False"
      }
}
