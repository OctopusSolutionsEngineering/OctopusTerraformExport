resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_an_aws_cloudformation_template" {
  name                  = "Deploy an AWS CloudFormation template"
  type                  = "Octopus.AwsRunCloudFormation"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Variable"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys an AWS CloudFormation template."
  package_requirement   = "LetOctopusDecide"
  slug                  = "deploy-an-aws-cloudformation-template"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Step.ConditionVariableExpression" = "#{Step.Run}"
      }
  execution_properties  = {
        "Octopus.Action.Aws.Region" = "us-east-2"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Aws.CloudFormationStackName" = "mystackname"
        "Octopus.Action.Aws.TemplateSource" = "Inline"
        "Octopus.Action.AwsAccount.Variable" = "Account.AWS"
        "Octopus.Action.AwsAccount.UseInstanceRole" = "False"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Aws.CloudFormationTemplateParameters" = jsonencode([])
        "Octopus.Action.Aws.CloudFormationTemplate" = "AWSTemplateFormatVersion: '2010-09-09'\nDescription: 'CloudFormation exports'\n \nConditions:\n  HasNot: !Equals [ 'true', 'false' ]\n \n# dummy (null) resource, never created\nResources:\n  NullResource:\n    Type: 'Custom::NullResource'\n    Condition: HasNot\n \nOutputs:\n  ExportsStackName:\n    Value: !Ref 'AWS::StackName'\n    Export:\n      Name: !Sub 'ExportsStackName-$${AWS::StackName}'"
        "Octopus.Action.Aws.AssumeRole" = "False"
        "Octopus.Action.Aws.WaitForCompletion" = "True"
      }
}
