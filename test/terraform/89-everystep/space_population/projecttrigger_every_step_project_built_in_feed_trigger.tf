resource "octopusdeploy_built_in_trigger" "projecttrigger_every_step_project_built_in_feed_trigger" {
  space_id                 = "${trimspace(var.octopus_space_id)}"
  channel_id               = "Channels-1"
  project_id               = "${octopusdeploy_project.project_every_step_project.id}"
  release_creation_package = { deployment_action = "Deploy a Package", package_reference = "" }
  depends_on               = [octopusdeploy_process_steps_order.process_step_order_every_step_project]
}
