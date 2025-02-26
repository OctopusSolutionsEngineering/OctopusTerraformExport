data "octopusdeploy_machine_policies" "default_machine_policy" {
  ids          = null
  partial_name = "Default Machine Policy"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_ssh_connection_worker" "optionals" {
  name              = "SSH Worker"
  machine_policy_id = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  worker_pool_ids = [octopusdeploy_static_worker_pool.workerpool_docker.id]
  account_id        = octopusdeploy_username_password_account.ssh_account.id
  host              = "hostname"
  port              = 22
  fingerprint       = "SHA256: 12345abc"
  dotnet_platform   = "linux-x64"
  proxy_id          = null
  is_disabled       = true
}