resource "octopusdeploy_channel" "backend_mainline" {
  name        = "Test"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  description = "Test channel"
  depends_on  = [octopusdeploy_project.deploy_frontend_project, octopusdeploy_process_steps_order.test]
  is_default  = true
}