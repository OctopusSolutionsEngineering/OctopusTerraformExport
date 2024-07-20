data "octopusdeploy_environments" "test" {
  ids          = []
  partial_name = "Test"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_tenant" "tenant_team_a" {
  name        = "Team A"
  description = "Test tenant"
  tenant_tags = ["tag1/a", "tag1/b"]
  depends_on = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b]


}

resource "octopusdeploy_tenant_project" "tenanta_project" {
  environment_ids = [data.octopusdeploy_environments.test.environments[0].id]
  project_id   = octopusdeploy_project.project_1.id
  tenant_id = octopusdeploy_tenant.tenant_team_a.id
}

resource "octopusdeploy_tenant" "tenant_team_b" {
  name        = "Team B"
  description = "Test tenant"
  tenant_tags = ["tag1/a", "tag1/b"]
  depends_on = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b]
}

resource "octopusdeploy_tenant_project" "tenantb_project" {
  environment_ids = [data.octopusdeploy_environments.test.environments[0].id]
  project_id   = octopusdeploy_project.project_2.id
  tenant_id    = octopusdeploy_tenant.tenant_team_b.id
}

resource "octopusdeploy_tenant" "tenant_team_c" {
  name        = "Team C"
  description = "Test tenant"
  tenant_tags = ["tag1/a", "tag1/b"]
  depends_on = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b]
}

resource "octopusdeploy_tenant_project" "tenantc_project" {
  environment_ids = [data.octopusdeploy_environments.test.environments[0].id]
  project_id   = octopusdeploy_project.project_2.id
  tenant_id = octopusdeploy_tenant.tenant_team_c.id
}