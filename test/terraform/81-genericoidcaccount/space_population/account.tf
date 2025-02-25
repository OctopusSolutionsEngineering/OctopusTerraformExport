resource "octopusdeploy_generic_oidc_account" "example" {
  name            = "Generic OpenID Connect"
  execution_subject_keys = ["space"]
  audience = "api://default"
  description = "A test account"
  tenanted_deployment_participation = "Untenanted"
}