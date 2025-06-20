variable "variable_ed8f23457693d0f390c1776381af53a97903d6786fa753de20da691c7e3cdbc8_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable OctopusPrintEvaluatedVariables"
  default     = "False"
}
resource "octopusdeploy_variable" "every_step_project_octopusprintevaluatedvariables_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_ed8f23457693d0f390c1776381af53a97903d6786fa753de20da691c7e3cdbc8_value}"
  name         = "OctopusPrintEvaluatedVariables"
  type         = "String"
  description  = "OctopusPrintEvaluatedVariables is a system variable that enbaled debugging by printing the evaluated value of each variable to the verbose logs. Set the variable to true to enable debugging."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
