resource "octopusdeploy_tenant" "tenant_team_a" {
  name        = "Team A"
  description = "Test tenant"
  tenant_tags = ["type/a", "type/b", "type/ignorethis"]
  depends_on  = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b, octopusdeploy_tag.tag_ignore]
}

resource "octopusdeploy_tenant_project" "team_a_frontend"{
  environment_ids = [
    octopusdeploy_environment.test_environment.id,
    octopusdeploy_environment.development_environment.id,
    octopusdeploy_environment.production_environment.id
  ]
  project_id = octopusdeploy_project.deploy_frontend_project.id
  tenant_id = octopusdeploy_tenant.tenant_team_a.id
}

resource "octopusdeploy_tenant" "tenant_excluded" {
  name        = "Excluded"
  description = "Excluded tenant"
  tenant_tags = ["type/excluded"]
  depends_on  = [octopusdeploy_tag.tag_excluded]

}

resource "octopusdeploy_tenant_project" "excluded_frontend" {
  environment_ids = [
    octopusdeploy_environment.test_environment.id,
    octopusdeploy_environment.development_environment.id,
    octopusdeploy_environment.production_environment.id
  ]
  project_id = octopusdeploy_project.deploy_frontend_project.id
  tenant_id = octopusdeploy_tenant.tenant_excluded.id
}