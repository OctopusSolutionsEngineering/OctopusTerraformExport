resource "octopusdeploy_token_account" "account_autopilot_service_account" {
  description                       = "A test account"
  name                              = "Token"
  environments                      = null
  tenant_tags                       = []
  tenants                           = [octopusdeploy_tenant.tenant_team_a.id]
  tenanted_deployment_participation = "Untenanted"
  token                             = "secretgoeshere"
}