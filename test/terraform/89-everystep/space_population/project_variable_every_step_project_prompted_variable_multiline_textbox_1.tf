variable "variable_92747481c7ef2d9942bfe8c6b3927d177a2d7e38f1cb6fb5d581aca0655d179b_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Prompted Variable Multiline TextBox"
  default     = "The default value\nover multiple lines"
}
resource "octopusdeploy_variable" "every_step_project_prompted_variable_multiline_textbox_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_92747481c7ef2d9942bfe8c6b3927d177a2d7e38f1cb6fb5d581aca0655d179b_value}"
  name         = "Prompted Variable Multiline TextBox"
  type         = "String"
  description  = "This is an example of a multiline textbox prompted variable"
  is_sensitive = false

  prompt {
    description = "This is the description"
    label       = "This is the label"
    is_required = false

    display_settings {
      control_type = "MultiLineText"
    }
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
