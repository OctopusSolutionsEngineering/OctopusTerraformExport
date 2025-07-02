resource "octopusdeploy_process" "process_every_step_project_example_runbook" {
  project_id = "${octopusdeploy_project.project_every_step_project.id}"
  runbook_id = "${octopusdeploy_runbook.runbook_every_step_project_example_runbook.id}"
  depends_on = []
}
