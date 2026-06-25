resource "octopusdeploy_channel" "ee_channel" {
  name        = "Features"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  description = "Feature branch channel"
  depends_on  = [octopusdeploy_project.deploy_frontend_project, octopusdeploy_process_steps_order.process_cloudformation_step]
  is_default  = true
  type = "EphemeralEnvironment"
  custom_field_definitions = [
    {
      field_name  = "Branch"
      "description" = "The branch name for the ephemeral environment"
    }
  ]
  ephemeral_environment_name_template = "#{Octopus.Release.CustomFields[Branch]}"
  parent_environment_id = octopusdeploy_parent_environment.example.id
}