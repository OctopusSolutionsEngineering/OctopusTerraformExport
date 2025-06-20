variable "variable_cbd3e5ec7fb4d4cb27ee477b5a70e6266048fc58b7470e1a21a725ed63a342e8_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Prompted Variable Dropdown List"
  default     = "Value1"
}
resource "octopusdeploy_variable" "every_step_project_prompted_variable_dropdown_list_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_cbd3e5ec7fb4d4cb27ee477b5a70e6266048fc58b7470e1a21a725ed63a342e8_value}"
  name         = "Prompted Variable Dropdown List"
  type         = "String"
  description  = "This is an example of a dropdown list prompted variable"
  is_sensitive = false

  prompt {
    description = "This is the description"
    label       = "This is the label"
    is_required = false

    display_settings {
      control_type = "Select"

      select_option {
        display_name = "Value1"
        value        = "Display text 1"
      }
      select_option {
        display_name = "Value2"
        value        = "Display text 2"
      }
    }
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
