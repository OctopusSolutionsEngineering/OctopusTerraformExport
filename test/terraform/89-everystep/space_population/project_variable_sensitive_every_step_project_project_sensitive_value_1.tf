variable "variable_d3944e50a3447d06f6fe89b13e55c7c0dd7a52a7d277d7160394fb71d60b315b_sensitive_value" {
  type        = string
  nullable    = true
  sensitive   = true
  description = "The secret variable value associated with the variable Project.Sensitive.Value"
  default     = "Change Me!"
}
resource "octopusdeploy_variable" "every_step_project_project_sensitive_value_1" {
  owner_id        = "${octopusdeploy_project.project_every_step_project.id}"
  name            = "Project.Sensitive.Value"
  type            = "Sensitive"
  description     = "This is a sensitive value. It includes the \"sensitive_value\" attribute. It does not include the \"value\" attribute."
  is_sensitive    = true
  sensitive_value = var.variable_d3944e50a3447d06f6fe89b13e55c7c0dd7a52a7d277d7160394fb71d60b315b_sensitive_value
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
