variable "variable_611e1a4c256ba09975397eab46b7a3e37587c93b77665826f272e428ce8ed8f2_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Runbook Scoped Variable"
  default     = "scoped to a runbook"
}
resource "octopusdeploy_variable" "every_step_project_runbook_scoped_variable_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_611e1a4c256ba09975397eab46b7a3e37587c93b77665826f272e428ce8ed8f2_value}"
  name         = "Runbook Scoped Variable"
  type         = "String"
  description  = "This is an example of a variable scoped to a runbook"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
