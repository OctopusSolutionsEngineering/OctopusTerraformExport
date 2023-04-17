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

  project_environment {
    environments = [data.octopusdeploy_environments.test.environments[0].id]
    project_id   = octopusdeploy_project.project_1.id
  }
}

resource "octopusdeploy_tenant" "tenant_team_b" {
  name        = "Team B"
  description = "Test tenant"
  tenant_tags = ["tag1/a", "tag1/b"]
  depends_on = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b]

  project_environment {
    environments = [data.octopusdeploy_environments.test.environments[0].id]
    project_id   = octopusdeploy_project.project_2.id
  }
}
