# https://github.com/OctopusDeploy/terraform-provider-octopusdeploy/issues/31
# GenericOidcAccount is not supported in the provider yet, so this is commented out.

# resource "octopusdeploy_variable" "every_step_project_project_oidc_account_1" {
#   owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
#   value        = "${octopusdeploy_aws_openid_connect_account.account_generic_oidc.id}"
#   name         = "Project.OIDC.Account"
#   type         = "GenericOidcAccount"
#   description  = "This variable points to a Generic OIDC account."
#   is_sensitive = false
#   lifecycle {
#     ignore_changes = [sensitive_value]
#   }
#   depends_on = []
# }
