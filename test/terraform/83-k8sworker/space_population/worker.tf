data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_kubernetes_agent_worker" "optionals" {
  name         = "K8s Worker"
  worker_pool_ids = [octopusdeploy_static_worker_pool.workerpool_docker.id]
  thumbprint   = "96203ED84246201C26A2F4360D7CBC36AC1D232D"
  uri          = "poll://kcxzcv2fpsxkn6tk9u6d/"
  machine_policy_id = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  upgrade_locked    = true
  is_disabled       = true
}