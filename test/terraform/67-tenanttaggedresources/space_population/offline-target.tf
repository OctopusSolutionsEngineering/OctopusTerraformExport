
resource "octopusdeploy_offline_package_drop_deployment_target" "target_offline" {
  applications_directory            = "c:\\temp"
  working_directory                 = "c:\\temp"
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "offline"
  roles                             = ["offline"]
  health_status                     = "Healthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type/a"]
  tenants                           = []
}