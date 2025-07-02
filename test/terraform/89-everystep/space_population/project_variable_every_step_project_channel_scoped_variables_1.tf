variable "variable_05ab8e7fa9ee7c179013575216c81bef1126bcd039b66508a096f9600bbc48c4_value" {
  type        = string
  nullable    = true
  sensitive   = false
  description = "The value associated with the variable Channel.Scoped.Variables"
  default     = "this is scoped to the hotfix channel"
}
resource "octopusdeploy_variable" "every_step_project_channel_scoped_variables_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${var.variable_05ab8e7fa9ee7c179013575216c81bef1126bcd039b66508a096f9600bbc48c4_value}"
  name         = "Channel.Scoped.Variables"
  type         = "String"
  description  = "This is an example of a variable scoped to a channel"
  is_sensitive = false

  scope {
    actions      = null
    channels     = ["${octopusdeploy_channel.channel_every_step_project_hotfix.id}"]
    environments = null
    machines     = null
    roles        = null
    tenant_tags  = null
  }
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
