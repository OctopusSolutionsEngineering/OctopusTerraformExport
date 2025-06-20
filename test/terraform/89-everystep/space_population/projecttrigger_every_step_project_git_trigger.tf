# resource "octopusdeploy_git_trigger" "projecttrigger_every_step_project_git_trigger" {
#   space_id    = "${trimspace(var.octopus_space_id)}"
#   name        = "Git Trigger"
#   description = "This is an example of a git trigger"
#   project_id  = "${octopusdeploy_project.project_every_step_project.id}"
#   channel_id  = "Channels-1"
#   sources     = []
#   depends_on  = [octopusdeploy_process.process_every_step_project]
# }
