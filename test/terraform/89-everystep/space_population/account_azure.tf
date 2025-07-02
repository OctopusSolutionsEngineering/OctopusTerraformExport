resource "octopusdeploy_azure_openid_connect" "account_azure" {
  name                              = "Azure"
  description                       = "An example of an unscoped Azure OIDC account available to all environments"
  environments                      = []
  tenant_tags                       = []
  tenants                           = []
  tenanted_deployment_participation = "Untenanted"
  subscription_id                   = "00000000-0000-0000-0000-000000000000"
  azure_environment                 = "AzureCloud"
  tenant_id                         = "00000000-0000-0000-0000-000000000000"
  application_id                    = "00000000-0000-0000-0000-000000000000"
  audience                          = "api://AzureADTokenExchange"
  account_test_subject_keys         = ["space"]
  execution_subject_keys            = ["space"]
  health_subject_keys               = ["space"]
  depends_on                        = []
  lifecycle {
    ignore_changes = []
  }
}
