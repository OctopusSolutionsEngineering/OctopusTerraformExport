variable "tenantvariable_e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855_value" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The value of the tenant project variable"
  default     = "A custom value for the Development environment"
}
resource "octopusdeploy_tenant_project_variable" "tenantprojectvariable_1_australian_office" {
  environment_id = "${octopusdeploy_environment.environment_development.id}"
  project_id     = "${octopusdeploy_project.project_every_step_project.id}"
  template_id    = "${octopusdeploy_project.project_every_step_project.template[0].id}"
  tenant_id      = "${octopusdeploy_tenant.tenant_australian_office.id}"
  value          = "${var.tenantvariable_e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855_value}"
  depends_on     = [octopusdeploy_tenant_project.tenant_project_australian_office_every_step_project]
}
