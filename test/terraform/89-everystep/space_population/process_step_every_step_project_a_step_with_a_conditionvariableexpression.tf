resource "octopusdeploy_process_step" "process_step_every_step_project_a_step_with_a_conditionvariableexpression" {
  name                  = "A step with a ConditionVariableExpression"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Variable"
  environments          = null
  excluded_environments = null
  notes                 = "This step uses a condition variable, which is defined in the \"Octopus.Step.ConditionVariableExpression\" property."
  package_requirement   = "LetOctopusDecide"
  slug                  = "a-step-with-a-conditionvariableexpression"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_id        = "${data.octopusdeploy_worker_pools.workerpool_hosted_windows.worker_pools[0].id}"
  properties            = {
        "Octopus.Step.ConditionVariableExpression" = "#{RunStep}"
      }
  execution_properties  = {
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "PowerShell"
        "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
        "OctopusUseBundledTooling" = "False"
      }
}
