resource "octopusdeploy_tenant" "tenant_team_a" {
  name        = "Team A"
  description = "Test tenant"
  tenant_tags = ["tag1/a", "tag1/b"]
  depends_on = [octopusdeploy_tag.tag_a, octopusdeploy_tag.tag_b]
}

resource octopus_tenant_project "tenanta_project" {
  environment_ids = [octopusdeploy_environment.test_environment.id, octopusdeploy_environment.development_environment.id, octopusdeploy_environment.production_environment.id]
  project_id   = octopusdeploy_project.deploy_frontend_project.id
  tenant_id = octopusdeploy_tenant.tenant_team_a.id
}

resource "octopusdeploy_tenant_common_variable" "tenantcommonvariable1_variablea" {
  library_variable_set_id = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.id}"
  template_id             = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.template[0].id}"
  tenant_id               = "${octopusdeploy_tenant.tenant_team_a.id}"
  value                   = "Override Variable A"
}

resource "octopusdeploy_tenant_common_variable" "tenantcommonvariable1_variableb" {
  library_variable_set_id = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.id}"
  template_id             = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.template[1].id}"
  tenant_id               = "${octopusdeploy_tenant.tenant_team_a.id}"
  value                   = "Override Variable B"
}

resource "octopusdeploy_tenant_common_variable" "tenantcommonvariable1_secret" {
  library_variable_set_id = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.id}"
  template_id             = "${octopusdeploy_library_variable_set.library_variable_set_variables_variableset.template[2].id}"
  tenant_id               = "${octopusdeploy_tenant.tenant_team_a.id}"
  value                   = "Override Secret"
}