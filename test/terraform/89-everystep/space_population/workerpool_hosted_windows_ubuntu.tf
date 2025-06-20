# These worker pools recreate those found in a hosted instance.
# Note that we use the suffix "Replacement" to not have Octoterra treat these as built in workerpools.
# If they were called "Hosted Ubuntu" and "Hosted Windows", Octoterra would assume the pools already exist in the target space.

resource "octopusdeploy_static_worker_pool" "ubuntu" {
  name        = "Hosted Ubuntu Replacement"
  description = "A test worker pool"
  is_default  = false
}

resource "octopusdeploy_static_worker_pool" "windows" {
  name        = "Hosted Windows Replacement"
  description = "A test worker pool"
  is_default  = false
}
