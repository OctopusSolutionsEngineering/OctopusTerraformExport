resource "octopusdeploy_aws_openid_connect_account" "account_generic_oidc" {
  name                              = "Generic OIDC"
  description                       = "An example of an unscoped generic OIDC account"
  environments                      = []
  tenant_tags                       = []
  tenants                           = []
  execution_subject_keys            = ["space"]
  tenanted_deployment_participation = "Untenanted"
  role_arn = "arn:aws:iam::123456789012:role/ExampleRole"
  depends_on                        = []
  lifecycle {
    ignore_changes = []
  }
}
