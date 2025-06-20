resource "octopusdeploy_gcp_account" "account_google_cloud_account" {
  name                              = "Google Cloud Account"
  description                       = "An example of a Google Cloud (GCP) Account scoped to the Development environment"
  environments                      = ["${octopusdeploy_environment.environment_development.id}"]
  tenant_tags                       = []
  tenants                           = []
  tenanted_deployment_participation = "Untenanted"
  json_key                          = "${var.account_google_cloud_account}"
  depends_on                        = []
  lifecycle {
    ignore_changes = [json_key]
  }
}
variable "account_google_cloud_account" {
  type        = string
  nullable    = false
  sensitive   = true
  description = "The GCP JSON key associated with the account Google Cloud Account"
  default     = "Change Me!"
}
