resource "octopusdeploy_process_step" "process_step_every_step_project_run_an_aws_cli_script" {
  name                  = "Run an AWS CLI Script"
  type                  = "Octopus.AwsRunScript"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This is an example script that run against AWS resources"
  package_requirement   = "LetOctopusDecide"
  slug                  = "run-an-aws-cli-script"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.AwsAccount.Variable" = "Account.AWS"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "PowerShell"
        "Octopus.Action.Aws.Region" = "us-east-1"
        "Octopus.Action.Aws.AssumeRole" = "False"
        "Octopus.Action.AwsAccount.UseInstanceRole" = "False"
        "Octopus.Action.Script.ScriptBody" = "echo \"Hi\""
        "OctopusUseBundledTooling" = "False"
      }
}
