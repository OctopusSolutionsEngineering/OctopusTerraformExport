resource "octopusdeploy_script_module" "library_variable_set_test2" {
  description = "Test script module"
  name        = "Script Module"

  script {
    body   = "echo \"hi\""
    syntax = "PowerShell"
  }
}
