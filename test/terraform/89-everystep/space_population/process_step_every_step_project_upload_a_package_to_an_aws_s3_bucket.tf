variable "project_every_step_project_step_upload_a_package_to_an_aws_s3_bucket_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Upload a package to an AWS S3 bucket in project Every Step Project"
  default     = "MyAWSPackage"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_upload_a_package_to_an_aws_s3_bucket" {
  name                  = "Upload a package to an AWS S3 bucket"
  type                  = "Octopus.AwsUploadS3"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step uploads a file to an AWS S3 bucket."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_upload_a_package_to_an_aws_s3_bucket_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "upload-a-package-to-an-aws-s3-bucket"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.AwsAccount.Variable" = "Account.AWS"
        "Octopus.Action.RunOnServer" = "false"
        "Octopus.Action.Aws.S3.PackageOptions" = jsonencode({
        "variableSubstitutionPatterns" = ""
        "structuredVariableSubstitutionPatterns" = ""
        "bucketKeyPrefix" = ""
        "storageClass" = "STANDARD"
        "cannedAcl" = "private"
        "tags" = []
        "bucketKey" = "mybucket"
        "bucketKeyBehaviour" = "Custom"
        "metadata" = []
                })
        "Octopus.Action.Aws.S3.BucketName" = "my-s3-bucket"
        "Octopus.Action.Aws.S3.TargetMode" = "EntirePackage"
        "Octopus.Action.Aws.AssumeRole" = "False"
        "Octopus.Action.Aws.Region" = "ap-southeast-2"
        "Octopus.Action.AwsAccount.UseInstanceRole" = "False"
      }
}
