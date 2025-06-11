resource "octopusdeploy_channel" "backend_mainline" {
  name        = "Test"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  description = "Test channel"
  depends_on  = [octopusdeploy_project.deploy_frontend_project, octopusdeploy_process_steps_order.process_cloudformation_step]
  is_default  = true
  rule {
    tag = "^$"
    action_package {
      deployment_action = "Test"
      package_reference = "test"
    }
  }
}