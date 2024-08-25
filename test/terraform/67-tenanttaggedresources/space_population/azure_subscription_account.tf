resource "octopusdeploy_azure_subscription_account" "account_subscription" {
  description                       = "A test account"
  name                              = "Subscription"
  environments                      = null
  tenants                           = null
  depends_on                        = [octopusdeploy_tag.tag_a]
  tenant_tags                       = ["type with space/a with space"]
  tenanted_deployment_participation = "Tenanted"
  storage_endpoint_suffix           = "storage_endpoint_suffix"
  subscription_id                   = "fde6a0ae-a1d4-40ae-91de-88f4ed898c03"
  azure_environment                 = "AzureCloud"
  management_endpoint               = "management_endpoint"
  certificate                       = file("dummycert2.txt")
}