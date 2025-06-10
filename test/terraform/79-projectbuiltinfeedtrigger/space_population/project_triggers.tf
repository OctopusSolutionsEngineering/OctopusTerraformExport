resource "octopusdeploy_built_in_trigger" "example" {
  project_id = octopusdeploy_project.deploy_frontend_project.id
  channel_id = octopusdeploy_channel.backend_mainline.id

  release_creation_package = {
    deployment_action = "step1"
    package_reference      = "package1"
  }

  depends_on = [octopusdeploy_process_steps_order.test]
}