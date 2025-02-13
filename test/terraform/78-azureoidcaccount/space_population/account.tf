resource "octopusdeploy_azure_openid_connect" "example" {
  application_id  = "10000000-0000-0000-0000-000000000000"
  name            = "Azure OIDC"
  description = "A test account"
  subscription_id = "20000000-0000-0000-0000-000000000000"
  tenant_id       = "30000000-0000-0000-0000-000000000000"
  execution_subject_keys = ["space"]
  health_subject_keys = ["space"]
  account_test_subject_keys = ["space"]
  audience = "api://AzureADTokenExchange"
}