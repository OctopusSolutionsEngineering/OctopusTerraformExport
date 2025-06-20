variable "variable_77a383b2349dcc39c966faa9e5ae72da855f6405e711813d52a6b78790fc6512_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Database.Password"
  default     = "development.database.internal"
}
resource "octopusdeploy_variable" "variables_example_variable_set_database_password_1" {
  owner_id     = "${octopusdeploy_library_variable_set.library_variable_set_variables_example_variable_set.id}"
  value        = "${var.variable_77a383b2349dcc39c966faa9e5ae72da855f6405e711813d52a6b78790fc6512_value}"
  name         = "Database.Password"
  type         = "String"
  description  = "This is an example of a variable scoped to an environment"
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
