resource "octopusdeploy_gcp_account" "account_google" {
  description                       = "A test account"
  name                              = "Google"
  environments                      = null
  tenants                           = null
  json_key                          = "secretgoeshere"
  tenanted_deployment_participation = "Tenanted"
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type with space/a with space"]
}