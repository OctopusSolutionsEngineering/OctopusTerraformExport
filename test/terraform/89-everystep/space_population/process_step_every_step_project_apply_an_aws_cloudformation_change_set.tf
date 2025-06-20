resource "octopusdeploy_process_step" "process_step_every_step_project_apply_an_aws_cloudformation_change_set" {
  name                  = "Apply an AWS CloudFormation Change Set"
  type                  = "Octopus.AwsApplyCloudFormationChangeSet"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Always"
  environments          = null
  excluded_environments = null
  notes                 = "This step applies an AWS CloudFormation Change Set."
  package_requirement   = "LetOctopusDecide"
  slug                  = "apply-an-aws-cloudformation-change-set"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Aws.CloudFormationStackName" = "mystack"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Aws.AssumeRole" = "False"
        "Octopus.Action.AwsAccount.Variable" = "Account.AWS"
        "Octopus.Action.AwsAccount.UseInstanceRole" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Aws.Region" = "ap-southeast-1"
        "Octopus.Action.Aws.CloudFormation.ChangeSet.Arn" = "mychangeset"
        "Octopus.Action.Aws.WaitForCompletion" = "True"
      }
}
