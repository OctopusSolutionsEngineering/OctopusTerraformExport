resource "octopusdeploy_deployment_freeze" "freeze" {
  name = "Xmas"
  start = "2024-12-25T00:00:00+10:00"
  end = "2024-12-27T00:00:00+08:00"
}

resource "octopusdeploy_deployment_freeze_project" "project_freeze" {
  deploymentfreeze_id = octopusdeploy_deployment_freeze.freeze.id
  project_id          = octopusdeploy_project.deploy_frontend_project.id
  environment_ids = [octopusdeploy_environment.development_environment.id]
}