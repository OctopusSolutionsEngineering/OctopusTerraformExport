resource "octopusdeploy_variable" "every_step_project_project_azure_account_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${octopusdeploy_azure_openid_connect.account_azure.id}"
  name         = "Project.Azure.Account"
  type         = "AzureAccount"
  description  = "This variable points to an Azure account."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
