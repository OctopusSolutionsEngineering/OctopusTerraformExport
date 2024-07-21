data "octopusdeploy_library_variable_sets" "variable" {
  partial_name = "Octopus Variables"
  skip         = 0
  take         = 1
}