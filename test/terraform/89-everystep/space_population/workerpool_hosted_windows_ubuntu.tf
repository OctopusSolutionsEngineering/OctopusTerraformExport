# These worker pools recreate those found in a hosted instance.

resource "octopusdeploy_static_worker_pool" "ubuntu" {
  name        = "Hosted Ubuntu"
  description = "A test worker pool"
  is_default  = false
}

resource "octopusdeploy_static_worker_pool" "windows" {
  name        = "Hosted Windows"
  description = "A test worker pool"
  is_default  = false
}
