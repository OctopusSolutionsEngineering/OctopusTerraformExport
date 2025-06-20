variable "variable_99b08078ba47238c5e4abfae90c644746cdea2cdeb685fbf8010dbf63ecd3563_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Step.Run"
  default     = "True"
}
resource "octopusdeploy_variable" "every_step_project_step_run_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_99b08078ba47238c5e4abfae90c644746cdea2cdeb685fbf8010dbf63ecd3563_value}"
  name         = "Step.Run"
  type         = "String"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
