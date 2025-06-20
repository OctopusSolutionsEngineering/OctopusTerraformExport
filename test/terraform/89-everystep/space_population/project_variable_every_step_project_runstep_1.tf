variable "variable_68d4ff7f1621b7cbf212371e9f137903a98148a9e91017d68bd46a636854459d_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable RunStep"
  default     = "true"
}
resource "octopusdeploy_variable" "every_step_project_runstep_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_68d4ff7f1621b7cbf212371e9f137903a98148a9e91017d68bd46a636854459d_value}"
  name         = "RunStep"
  type         = "String"
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
