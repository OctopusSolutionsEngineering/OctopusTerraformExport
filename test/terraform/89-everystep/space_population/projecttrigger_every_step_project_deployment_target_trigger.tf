resource "octopusdeploy_project_deployment_target_trigger" "projecttrigger_every_step_project_deployment_target_trigger" {
  name             = "Deployment Target Trigger"
  project_id       = "${octopusdeploy_project.project_every_step_project.id}"
  event_categories = ["MachineAdded", "MachineDeploymentRelatedPropertyWasUpdated", "MachineCleanupFailed"]
  environment_ids  = [octopusdeploy_environment.environment_development.id]
  event_groups     = ["Machine", "MachineCritical", "MachineAvailableForDeployment", "MachineUnavailableForDeployment", "MachineHealthChanged"]
  roles            = ["Kubernetes"]
  should_redeploy  = false
}
