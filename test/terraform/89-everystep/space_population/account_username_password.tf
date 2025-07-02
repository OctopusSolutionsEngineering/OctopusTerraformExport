resource "octopusdeploy_username_password_account" "account_username_password" {
  name                              = "Username Password"
  description                       = "An example of a username password account scoped to the Production environment"
  environments                      = ["${octopusdeploy_environment.environment_production.id}"]
  tenant_tags                       = []
  tenants                           = []
  tenanted_deployment_participation = "Untenanted"
  username                          = "username"
  password                          = "${var.account_username_password}"
  depends_on                        = []
  lifecycle {
    ignore_changes = [password]
  }
}
variable "account_username_password" {
  type        = string
  nullable    = false
  sensitive   = true
  description = "The password associated with the account Username Password"
  default     = "Change Me!"
}
