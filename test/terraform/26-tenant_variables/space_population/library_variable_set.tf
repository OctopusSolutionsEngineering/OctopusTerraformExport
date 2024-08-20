resource "octopusdeploy_library_variable_set" "library_variable_set_octopus_variables" {
  name        = "Octopus Variables"

  template {
    name             = "template"
    label            = "a"
    help_text        = "a"
    default_value    = "a"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }

  template {
    name             = "Drop Down"
    label            = "Label"
    help_text        = "Help Text"
    default_value    = "Value1"
    display_settings = { "Octopus.ControlType" = "Select", "Octopus.SelectOptions" = "Value1|Display text 1\nValue2|Display text 2" }
  }

  template {
    name             = "Check Box"
    label            = "Check Box"
    help_text        = "Help Text"
    default_value    = "Default"
    display_settings = { "Octopus.ControlType" = "Checkbox" }
  }
}
