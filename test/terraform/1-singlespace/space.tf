resource "octopusdeploy_space" "octopus_space_test" {
  name                  = "${var.octopus_space_test_name}"
  is_default            = false
  is_task_queue_stopped = false
  space_managers_teams  = ["teams-administrators"]
}

output "octopus_space_id" {
  value = octopusdeploy_space.octopus_space_test.id
}

variable "octopus_space_test_name" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The name of the new space"
  default     = "Test"
}