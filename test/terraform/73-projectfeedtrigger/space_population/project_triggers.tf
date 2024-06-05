resource "octopusdeploy_external_feed_create_release_trigger" "my_trigger" {
  name       = "My feed trigger"
  space_id   = var.octopus_space_id
  project_id = octopusdeploy_project.deploy_frontend_project.id
  channel_id = octopusdeploy_channel.backend_mainline.id
  package {
    deployment_action_slug = "step1"
    package_reference      = "package1"
  }
  depends_on = [octopusdeploy_deployment_process.test]
}