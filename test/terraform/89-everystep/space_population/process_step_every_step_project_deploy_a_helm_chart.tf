variable "project_every_step_project_step_deploy_a_helm_chart_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy a Helm Chart in project Every Step Project"
  default     = "MyHelmApp"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_a_helm_chart" {
  name                  = "Deploy a Helm Chart"
  type                  = "Octopus.HelmChartUpgrade"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys a Kubernetes Helm chart."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${octopusdeploy_helm_feed.feed_helm.id}", id = null, package_id = "${var.project_every_step_project_step_deploy_a_helm_chart_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-a-helm-chart"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "Kubernetes"
      }
  execution_properties  = {
        "Octopus.Action.Kubernetes.ResourceStatusCheck" = "True"
        "Octopus.Action.Script.ScriptSource" = "Package"
        "Octopus.Action.Helm.ResetValues" = "True"
        "Octopus.Action.Helm.Namespace" = "mycustomnamespace"
      }
}
