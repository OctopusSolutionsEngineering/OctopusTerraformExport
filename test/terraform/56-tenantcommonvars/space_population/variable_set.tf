resource "octopusdeploy_library_variable_set" "library_variable_set_variables_variableset" {
  name        = "VariableSet"
  description = ""

  template {
    name             = "VariableA"
    label            = ""
    default_value    = "a"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }
  template {
    name             = "VariableB"
    label            = ""
    default_value    = "b"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }
  template {
    name             = "secret"
    label            = ""
    default_value    = ""
    display_settings = { "Octopus.ControlType" = "Sensitive" }
  }
}