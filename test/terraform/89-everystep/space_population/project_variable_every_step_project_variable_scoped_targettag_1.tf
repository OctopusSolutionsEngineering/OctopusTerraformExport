variable "variable_d45752628c19f90595dc15b0641af818a054481ab277739ab64b6325329d9797_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Variable.Scoped.TargetTag"
  default     = "scoped to target"
}
resource "octopusdeploy_variable" "every_step_project_variable_scoped_targettag_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_d45752628c19f90595dc15b0641af818a054481ab277739ab64b6325329d9797_value}"
  name         = "Variable.Scoped.TargetTag"
  type         = "String"
  description  = "This is an example of a variable scoped to a target tag (or role)"
  is_sensitive = false

  scope {
    actions      = null
    channels     = null
    environments = null
    machines     = null
    roles        = ["Kubernetes"]
    tenant_tags  = null
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
