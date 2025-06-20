variable "variable_c9c5116a122c3d94886131532de205ba0ad9c0b41280a670fca888b7689cd427_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Database.Password"
  default     = "production.database.internal"
}
resource "octopusdeploy_variable" "variables_example_variable_set_database_password_3" {
  owner_id     = "${octopusdeploy_library_variable_set.library_variable_set_variables_example_variable_set.id}"
  value        = "${var.variable_c9c5116a122c3d94886131532de205ba0ad9c0b41280a670fca888b7689cd427_value}"
  name         = "Database.Password"
  type         = "String"
  description  = "This is an example of a variable scoped to an environment"
  is_sensitive = false

  scope {
    actions      = null
    channels     = null
    environments = ["${octopusdeploy_environment.environment_production.id}"]
    machines     = null
    roles        = null
    tenant_tags  = null
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
