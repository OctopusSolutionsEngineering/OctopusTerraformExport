variable "variable_4b088a7c2727493aa87f9feb95333a19db475b66de20ad90477ea8cffe33d5dc_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable OctopusPrintVariables"
  default     = "False"
}
resource "octopusdeploy_variable" "every_step_project_octopusprintvariables_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_4b088a7c2727493aa87f9feb95333a19db475b66de20ad90477ea8cffe33d5dc_value}"
  name         = "OctopusPrintVariables"
  type         = "String"
  description  = "OctopusPrintVariables is a system variable that enables debugging by printing the value of each variable to the verbose logs. Set the value to true to enable debugging."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
