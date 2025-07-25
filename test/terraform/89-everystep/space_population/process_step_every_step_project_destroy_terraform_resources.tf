resource "octopusdeploy_process_step" "process_step_every_step_project_destroy_terraform_resources" {
  name                  = "Destroy Terraform resources"
  type                  = "Octopus.TerraformDestroy"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = ["${octopusdeploy_environment.environment_development.id}"]
  excluded_environments = null
  notes                 = "This step destroys resources created by a Terraform configuration file."
  package_requirement   = "LetOctopusDecide"
  slug                  = "destroy-terraform-resources"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
      }
  execution_properties  = {
        "Octopus.Action.Terraform.Template" = "ariable \"images\" {\n  type = \"map\"\n\n  default = {\n    us-east-1 = \"image-1234\"\n    us-west-2 = \"image-4567\"\n  }\n}\n\nvariable \"test2\" {\n  type    = \"map\"\n  default = {\n    val1 = [\"hi\"]\n  }\n}\n\nvariable \"test3\" {\n  type    = \"map\"\n  default = {\n    val1 = {\n      val2 = \"hi\"\n    }\n  }\n}\n\nvariable \"test4\" {\n  type    = \"map\"\n  default = {\n    val1 = {\n      val2 = [\"hi\"]\n    }\n  }\n}\n\n# Example of getting an element from a list in a map\noutput \"nestedlist\" {\n  value = \"$${element(var.test2[\"val1\"], 0)}\"\n}\n\n# Example of getting an element from a nested map\noutput \"nestedmap\" {\n  value = \"$${lookup(var.test3[\"val1\"], \"val2\")}\"\n}"
        "Octopus.Action.Terraform.ManagedAccount" = "None"
        "Octopus.Action.Terraform.GoogleCloudAccount" = "False"
        "Octopus.Action.Terraform.TemplateParameters" = jsonencode({
        "test2" = "{\n  val1 = [\n    \"hi\"\n  ]\n}"
        "test3" = "{\n  val1 = {\n    val2 = \"hi\"\n  }\n}"
        "test4" = "{\n  val1 = {\n    val2 = [\n      \"hi\"\n    ]\n  }\n}"
                })
        "Octopus.Action.Terraform.AllowPluginDownloads" = "True"
        "Octopus.Action.Terraform.AzureAccount" = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Terraform.PlanJsonOutput" = "False"
        "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "True"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Terraform.RunAutomaticFileSubstitution" = "True"
      }
}
