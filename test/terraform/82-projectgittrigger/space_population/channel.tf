resource "octopusdeploy_channel" "backend_mainline" {
  name        = "Test"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  description = "Test channel"
  depends_on  = [octopusdeploy_project.deploy_frontend_project, octopusdeploy_deployment_process.test]
  is_default  = true
}