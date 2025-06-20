resource "octopusdeploy_variable" "every_step_project_project_usernamepassword_account_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${octopusdeploy_username_password_account.account_username_password.id}"
  name         = "Project.UsernamePassword.Account"
  type         = "UsernamePasswordAccount"
  description  = "This variable points to a username/password account."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
