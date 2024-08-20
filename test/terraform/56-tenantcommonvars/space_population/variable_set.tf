resource "octopusdeploy_library_variable_set" "library_variable_set_variables_variableset" {
  name        = "VariableSet"
  description = ""

  template {
    name             = "VariableA"
    label            = "This is a variable"
    default_value    = "a"
    help_text = "This is help text"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }
  template {
    name             = "VariableB"
    label            = "This is a variable"
    default_value    = "b"
    help_text = "This is help text"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }
  template {
    name             = "secret"
    label            = "This is a secret variable"
    default_value    = "a secret"
    help_text = "This is help text"
    display_settings = { "Octopus.ControlType" = "Sensitive" }
  }
}