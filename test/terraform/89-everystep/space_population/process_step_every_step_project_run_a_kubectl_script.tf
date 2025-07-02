resource "octopusdeploy_process_step" "process_step_every_step_project_run_a_kubectl_script" {
  name                  = "Run a kubectl script"
  type                  = "Octopus.KubernetesRunScript"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This runs a custom script using kubectl within the context of a Kubernetes cluster."
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-a-kubectl-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "Kubernetes"
      }
  execution_properties  = {
        "Octopus.Action.KubernetesContainers.Namespace" = "mynamespace"
        "Octopus.Action.RunOnServer" = "true"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "PowerShell"
        "Octopus.Action.Script.ScriptBody" = "Write-Host \"Hello World\""
      }
}
