resource "octopusdeploy_variable" "every_step_project_account_aws_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${octopusdeploy_aws_openid_connect_account.account_aws_oidc.id}"
  name         = "Account.AWS"
  type         = "AmazonWebServicesAccount"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
