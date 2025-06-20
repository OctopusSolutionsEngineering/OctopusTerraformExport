resource "octopusdeploy_variable" "every_step_project_project_gcp_account_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${octopusdeploy_gcp_account.account_google_cloud_account.id}"
  name         = "Project.GCP.Account"
  type         = "GoogleCloudAccount"
  description  = "This variable points to a Google Cloud Account."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
