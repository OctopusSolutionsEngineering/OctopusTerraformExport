resource "octopusdeploy_azure_service_fabric_cluster_deployment_target" "target_service_fabric" {
  environments                      = ["${octopusdeploy_environment.development_environment.id}"]
  name                              = "Service Fabric"
  roles                             = ["cloud"]
  connection_endpoint               = "http://endpoint"
  aad_client_credential_secret      = ""
  aad_credential_type               = "UserCredential"
  aad_user_credential_password      = "${var.target_service_fabric}"
  aad_user_credential_username      = "username"
  certificate_store_location        = ""
  certificate_store_name            = ""
  client_certificate_variable       = ""
  health_status                     = "Unhealthy"
  is_disabled                       = false
  machine_policy_id                 = "${data.octopusdeploy_machine_policies.default_machine_policy.machine_policies[0].id}"
  shell_name                        = "Unknown"
  shell_version                     = "Unknown"
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type/a"]
  tenants                           = []
  thumbprint                        = ""
}
variable "target_service_fabric" {
  type        = string
  nullable    = true
  sensitive   = true
  description = "The secret variable value associated with the target \"Service Fabric\""
}