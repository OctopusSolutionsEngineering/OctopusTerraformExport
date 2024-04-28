resource "octopusdeploy_external_feed_create_release_trigger" "my_trigger" {
  name        = "My feed trigger"
  space_id = var.octopus_space_id
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  package {
    deployment_action = "Get MySQL Host"
    package_reference = "package1"
  }
}