resource "octopusdeploy_aws_account" "account_aws_account" {
  name                              = "AWS Account"
  description                       = ""
  environments                      = [octopusdeploy_environment.test_environment.id]
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  access_key                        = "ABCDEFGHIJKLMNOPQRST"
  secret_key                        = "secretgoeshere"
}
