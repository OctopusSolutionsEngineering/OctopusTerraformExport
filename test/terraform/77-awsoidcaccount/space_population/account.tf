resource "octopusdeploy_aws_openid_connect_account" "awsoidc" {
  description                       = "A test account"
  name                              = "AWSOIDC"
  environments                      = null
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  account_test_subject_keys         = ["space"]
  execution_subject_keys = ["space"]
  health_subject_keys = ["space"]
  session_duration = 3600
  role_arn = "whatever"
}