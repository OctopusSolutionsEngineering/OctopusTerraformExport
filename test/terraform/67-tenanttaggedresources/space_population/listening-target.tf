resource "octopusdeploy_listening_tentacle_deployment_target" "target_vm_listening_ngrok" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "listening"
  roles                             = ["vm"]
  tentacle_url                      = "https://tentacle/"
  thumbprint                        = "55E05FD1B0F76E60F6DA103988056CE695685FD1"
  is_disabled                       = false
  is_in_process                     = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type/a"]
  tenants                           = []

  tentacle_version_details {
  }
}