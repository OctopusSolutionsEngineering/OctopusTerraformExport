variable "variable_3d056a15b43fe31497140b530622f557fb61b7880e317889a9b7f83ed41b4b7a_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Prompted Variable TextBox"
  default     = "The default value"
}
resource "octopusdeploy_variable" "every_step_project_prompted_variable_textbox_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_3d056a15b43fe31497140b530622f557fb61b7880e317889a9b7f83ed41b4b7a_value}"
  name         = "Prompted Variable TextBox"
  type         = "String"
  description  = "This is an example of a single line textbox prompted variable"
  is_sensitive = false

  prompt {
    description = "This is the description"
    label       = "This is the label"
    is_required = false

    display_settings {
      control_type = "SingleLineText"
    }
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
