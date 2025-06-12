data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_listening_tentacle_worker" "optionals" {
  name              = "Listening Worker"
  machine_policy_id = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  worker_pool_ids = [octopusdeploy_static_worker_pool.workerpool_docker.id]
  thumbprint        = "96203ED84246201C26A2F4360D7CBC36AC1D232D"
  uri               = "https://tentacle.listening/"
  is_disabled       = true

}