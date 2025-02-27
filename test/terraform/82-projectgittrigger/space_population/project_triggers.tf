resource "octopusdeploy_git_trigger" "my_trigger" {
  name                = "My Git trigger"
  description         = "My Git trigger description"
  project_id          = octopusdeploy_project.deploy_frontend_project.id
  channel_id          = octopusdeploy_channel.backend_mainline.id
  sources = [{
    deployment_action_slug = "step1"
    git_dependency_name    = ""
    include_file_paths     = [
      "include/me",
      "include/this/too"
    ]
    exclude_file_paths     = [
      "exclude/me",
      "exclude/this/too"
    ]
  }]
}