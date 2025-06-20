data "octopusdeploy_worker_pools" "workerpool_hosted_ubuntu" {
  ids          = null
  partial_name = "Hosted Ubuntu"
  skip         = 0
  take         = 1
  lifecycle {
    postcondition {
      error_message = "Failed to resolve a worker pool called \"Hosted Ubuntu\". This resource must exist in the space before this Terraform configuration is applied."
      condition     = length(self.worker_pools) != 0
    }
  }
}
