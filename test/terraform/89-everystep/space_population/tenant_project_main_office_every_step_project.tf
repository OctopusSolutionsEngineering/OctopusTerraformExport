resource "octopusdeploy_tenant_project" "tenant_project_main_office_every_step_project" {
  tenant_id       = "${octopusdeploy_tenant.tenant_main_office.id}"
  project_id      = "${octopusdeploy_project.project_every_step_project.id}"
  environment_ids = ["${octopusdeploy_environment.environment_development.id}", "${octopusdeploy_environment.environment_test.id}", "${octopusdeploy_environment.environment_production.id}", "${octopusdeploy_environment.environment_security.id}"]
}
