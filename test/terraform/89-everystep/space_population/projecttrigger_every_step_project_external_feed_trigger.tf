resource "octopusdeploy_external_feed_create_release_trigger" "projecttrigger_every_step_project_external_feed_trigger" {
  space_id   = "${trimspace(var.octopus_space_id)}"
  project_id = "${octopusdeploy_project.project_every_step_project.id}"
  name       = "External Feed Trigger"
  channel_id = "Channels-1"

  package {
    deployment_action_slug = "deploy-a-helm-chart"
    package_reference      = ""
  }
  depends_on = [octopusdeploy_process_steps_order.process_step_order_every_step_project]
}
