resource "octopusdeploy_process_step" "process_step_every_step_project_delete_an_aws_cloudformation_stack" {
  name                  = "Delete an AWS CloudFormation stack"
  type                  = "Octopus.AwsDeleteCloudFormation"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Failure"
  environments          = null
  excluded_environments = null
  notes                 = "This step deletes an AWS CloudFormation stack."
  package_requirement   = "LetOctopusDecide"
  slug                  = "delete-an-aws-cloudformation-stack"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Aws.Region" = "us-east-2"
        "Octopus.Action.Aws.CloudFormationStackName" = "my-stack-name"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Aws.WaitForCompletion" = "True"
        "Octopus.Action.Aws.AssumeRole" = "False"
        "Octopus.Action.AwsAccount.UseInstanceRole" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.AwsAccount.Variable" = "Account.AWS"
      }
}
