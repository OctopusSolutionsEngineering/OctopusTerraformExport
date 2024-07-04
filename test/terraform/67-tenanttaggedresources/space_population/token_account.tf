resource "octopusdeploy_token_account" "account_autopilot_service_account" {
  description                       = "A test account"
  name                              = "Token"
  environments                      = null
  tenants                           = null
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type with space/a with space"]
  token                             = "secretgoeshere"
}