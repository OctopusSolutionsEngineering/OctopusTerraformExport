# resource "octopusdeploy_tenant_common_variable" "tenantcommonvariable3_australian_office" {
#   library_variable_set_id = "${octopusdeploy_library_variable_set.library_variable_set_variables_example_variable_set.id}"
#   template_id             = "${octopusdeploy_library_variable_set.library_variable_set_variables_example_variable_set.template[9].id}"
#   tenant_id               = "${octopusdeploy_tenant.tenant_australian_office.id}"
#   value                   = "single line of text"
#   depends_on              = [octopusdeploy_tenant_project.tenant_project_australian_office_every_step_project]
# }
