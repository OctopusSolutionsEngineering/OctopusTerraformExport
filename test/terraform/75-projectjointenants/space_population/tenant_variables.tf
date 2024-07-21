resource "octopusdeploy_tenant_project_variable" "tenantprojectvariable6_team_a" {
  environment_id = data.octopusdeploy_environments.dev.environments[0].id
  project_id     = octopusdeploy_project.deploy_frontend_project.id
  template_id    = octopusdeploy_project.deploy_frontend_project.template[0].id
  tenant_id      = data.octopusdeploy_tenants.tenant.tenants[0].id
  value          = "my project variable"
  depends_on = [octopusdeploy_tenant_project.tenant_project]
}

resource "octopusdeploy_tenant_common_variable" "tenantcommonvariable1_team_a" {
  library_variable_set_id = data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].id
  template_id             = data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].template[0].id
  tenant_id               = data.octopusdeploy_tenants.tenant.tenants[0].id
  value                   = "my common variable"
  depends_on = [octopusdeploy_tenant_project.tenant_project]
}
