resource "octopusdeploy_variable" "every_step_project_project_workerpool_1" {
  owner_id     = "${octopusdeploy_project.project_every_step_project.id}"
  value        = "${octopusdeploy_static_worker_pool.workerpool_worker_pool.id}"
  name         = "Project.WorkerPool"
  type         = "WorkerPool"
  description  = "This variable points to a worker pool."
  is_sensitive = false
  lifecycle {
    ignore_changes = [sensitive_value]
  }
  depends_on = []
}
