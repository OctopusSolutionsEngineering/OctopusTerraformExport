resource "octopusdeploy_static_worker_pool" "static_pool" {
  description = "A worker pool"
  is_default  = true
  name        = "Static pool"
  sort_order  = 5
}