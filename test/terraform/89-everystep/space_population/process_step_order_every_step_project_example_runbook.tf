resource "octopusdeploy_process_steps_order" "process_step_order_every_step_project_example_runbook" {
  process_id = "${octopusdeploy_process.process_every_step_project_example_runbook.id}"
  steps      = ["${octopusdeploy_process_step.process_step_every_step_project_example_runbook_run_a_script.id}"]
}
