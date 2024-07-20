resource "octopusdeploy_tenant" "tenant_team_a" {
  name        = "Team A"
  description = "Test tenant"


}

resource "octopusdeploy_tenant_project" "tenant_project" {
  environments = [octopusdeploy_environment.test_environment.id, octopusdeploy_environment.development_environment.id, octopusdeploy_environment.production_environment.id]
  project_id   = octopusdeploy_project.deploy_frontend_project.id
  tenant_id    = octopusdeploy_tenant.tenant_team_a.id
}