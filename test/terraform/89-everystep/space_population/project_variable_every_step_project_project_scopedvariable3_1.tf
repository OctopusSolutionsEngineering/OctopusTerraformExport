variable "variable_b3c310414023885b86945214cc5049e9a5e65faf7afe5009283d998a283e072c_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Project.ScopedVariable3"
  default     = "deployment process scoped variable"
}
resource "octopusdeploy_variable" "every_step_project_project_scopedvariable3_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_b3c310414023885b86945214cc5049e9a5e65faf7afe5009283d998a283e072c_value}"
  name         = "Project.ScopedVariable3"
  type         = "String"
  description  = "This variable is scoped to the deployment process"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
