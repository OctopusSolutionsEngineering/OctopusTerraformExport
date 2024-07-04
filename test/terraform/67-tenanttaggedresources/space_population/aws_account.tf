resource "octopusdeploy_aws_account" "account_aws_account" {
  name                              = "AWS Account"
  description                       = ""
  environments                      = null
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type with space/a with space"]
  tenants                           = null
  tenanted_deployment_participation = "Tenanted"
  access_key                        = "ABCDEFGHIJKLMNOPQRST"
  secret_key                        = "secretgoeshere"
}
