resource "octopusdeploy_static_worker_pool" "workerpool_worker_pool" {
  name        = "Worker Pool"
  description = "An example of a worker pool"
  is_default  = false
}
