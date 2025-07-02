variable "variable_4ec6443f5925a2d26a4a0f9f679bcf5a3e3d312dec322178456e41d33aaaef24_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Scoped.To.Step"
  default     = "whatever"
}
resource "octopusdeploy_variable" "every_step_project_scoped_to_step_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_4ec6443f5925a2d26a4a0f9f679bcf5a3e3d312dec322178456e41d33aaaef24_value}"
  name         = "Scoped.To.Step"
  type         = "String"
  description  = "This is an example of a variable scoped to a step"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
