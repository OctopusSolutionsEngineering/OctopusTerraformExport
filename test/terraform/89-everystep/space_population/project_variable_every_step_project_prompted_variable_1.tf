variable "variable_e77104f1f9e3545fd603c3c969af1adba946d94ef1e4b183f3eeed5d9000f0ee_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Prompted Variable"
  default     = ""
}
resource "octopusdeploy_variable" "every_step_project_prompted_variable_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_e77104f1f9e3545fd603c3c969af1adba946d94ef1e4b183f3eeed5d9000f0ee_value}"
  name         = "Prompted Variable"
  type         = "String"
  description  = "This is an example of a checkbox prompted variable"
  is_sensitive = false

  prompt {
    description = "This is the description"
    label       = "This is the label"
    is_required = false

    display_settings {
      control_type = "Checkbox"
    }
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
