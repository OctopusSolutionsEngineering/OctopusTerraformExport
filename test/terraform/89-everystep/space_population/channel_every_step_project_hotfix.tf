resource "octopusdeploy_channel" "channel_every_step_project_hotfix" {
  name        = "Hotfix"
  description = "This is an example channel with package version rules"
  project_id  = "${octopusdeploy_project.project_every_step_project.id}"
  is_default  = false

  rule {

    action_package {
      deployment_action = "Deploy a Helm Chart"
    }

    tag           = "^featurebranch$"
    version_range = "[1.0,)"
  }

  tenant_tags = []
  depends_on  = [octopusdeploy_process_steps_order.process_step_order_every_step_project,octopusdeploy_process_steps_order.process_step_order_every_step_project_example_runbook]
}
