variable "variable_5c86a8c0abd23511082ab508a62306c21e821633e511a3a020bceb37c997b949_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Project.ScopedVariable1"
  default     = "scoped variable"
}
resource "octopusdeploy_variable" "every_step_project_project_scopedvariable1_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_5c86a8c0abd23511082ab508a62306c21e821633e511a3a020bceb37c997b949_value}"
  name         = "Project.ScopedVariable1"
  type         = "String"
  description  = "This variable is scoped to an environment"
  is_sensitive = false

  scope {
    actions      = null
    channels     = null
    environments = ["${octopusdeploy_environment.environment_development.id}"]
    machines     = null
    roles        = null
    tenant_tags  = null
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
