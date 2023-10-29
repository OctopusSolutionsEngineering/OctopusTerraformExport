resource "octopusdeploy_username_password_account" "account_gke" {
  description                       = "A test account"
  name                              = "UsernamePasswordAccount"
  environments                      = null
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type/a"]
  tenanted_deployment_participation = "Tenanted"
  tenants                           = null
  username                          = "admin"
  password                          = "secretgoeshere"
}