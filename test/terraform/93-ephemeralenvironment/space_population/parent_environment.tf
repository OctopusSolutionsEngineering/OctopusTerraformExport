resource "octopusdeploy_parent_environment" "example" {
  name                          = "Parent Environment"
  space_id                      = var.octopus_space_id
  description                   = "A parent environment."
  use_guided_failure            = false
  automatic_deprovisioning_rule = {
    days = 7
    hours = 12
  }
}