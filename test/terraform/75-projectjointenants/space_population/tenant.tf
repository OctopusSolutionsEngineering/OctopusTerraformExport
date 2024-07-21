data "octopusdeploy_tenants" "tenant" {
  ids          = null
  partial_name = "Team A"
  skip         = 0
  take         = 1
}

data "octopusdeploy_environments" "test" {
  ids          = []
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_environments" "dev" {
  ids          = []
  partial_name = "Development"
  skip         = 0
  take         = 1
}

data "octopusdeploy_environments" "prod" {
  ids          = []
  partial_name = "Production"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_tenant_project" "tenant_project" {
  environment_ids = [data.octopusdeploy_environments.dev.environments[0].id,
    data.octopusdeploy_environments.test.environments[0].id,
    data.octopusdeploy_environments.prod.environments[0].id]
  project_id   = octopusdeploy_project.deploy_frontend_project.id
  tenant_id    = data.octopusdeploy_tenants.tenant.tenants[0].id
}