data "octopusdeploy_feeds" "project_feed" {
  feed_type    = "OctopusProject"
  skip         = 0
  take         = 1
}

data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

data "octopusdeploy_project_groups" "project_group" {
  ids          = null
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_feeds" "built_in_feed" {
  feed_type    = "BuiltIn"
  ids          = null
  partial_name = ""
  skip         = 0
  take         = 1
}

data "octopusdeploy_git_credentials" "git" {
  name = "matt"
  skip = 0
  take = 1
}

data "octopusdeploy_library_variable_sets" "variable" {
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_library_variable_sets" "variable2" {
  partial_name = "Test2"
  skip         = 0
  take         = 1
}

data "octopusdeploy_library_variable_sets" "variable3" {
  partial_name = "Test3"
  skip         = 0
  take         = 1
}

data "octopusdeploy_worker_pools" "workerpool_default" {
  name = "Default Worker Pool"
  ids  = null
  skip = 0
  take = 1
}


data "octopusdeploy_worker_pools" "worker_pool_docker" {
  ids          = null
  partial_name = "Docker"
  skip         = 0
  take         = 1
}

data "octopusdeploy_feeds" "feed_octopus_server_releases__built_in_" {
  feed_type    = "OctopusProject"
  ids          = null
  partial_name = ""
  skip         = 0
  take         = 1
  space_id = var.octopus_space_id
  lifecycle {
    postcondition {
      error_message = "Failed to resolve a feed called \"Octopus Server Releases (built-in)\". This resource must exist in the space before this Terraform configuration is applied."
      condition     = length(self.feeds) != 0
    }
  }
}

resource "octopusdeploy_project" "project_1" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test"
  project_group_id                     = data.octopusdeploy_project_groups.project_group.project_groups[0].id
  tenanted_deployment_participation    = "Untenanted"
  included_library_variable_sets       = [
    data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].id,
    data.octopusdeploy_library_variable_sets.variable2.library_variable_sets[0].id,
    data.octopusdeploy_library_variable_sets.variable3.library_variable_sets[0].id
  ]
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
}

resource "octopusdeploy_process" "deployment_process_project_one" {
  project_id = "${octopusdeploy_project.project_1.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_project_one" {
  process_id = "${octopusdeploy_process.deployment_process_project_one.id}"
  steps      = ["${octopusdeploy_process_step.process_step_project_one.id}"]
}

resource "octopusdeploy_process_step" "process_step_project_one" {
  name                  = "Deploy a Release"
  type                  = "Octopus.DeployRelease"
  process_id            = "${octopusdeploy_process.deployment_process_project_one.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  primary_package       = {
    acquisition_location = "NotAcquired",
    feed_id = data.octopusdeploy_feeds.feed_octopus_server_releases__built_in_.feeds[0].id,
    id = null,
    package_id = data.octopusdeploy_projects.other.projects[0].id,
    properties = null }
  slug                  = "deploy-a-release"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_id        = data.octopusdeploy_worker_pools.worker_pool_docker.worker_pools[0].id
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.DeployRelease.DeploymentCondition" = "Always"
    "Octopus.Action.DeployRelease.ProjectId" = data.octopusdeploy_projects.other.projects[0].id
    "Octopus.Action.RunOnServer" = "true"
  }
}

resource "octopusdeploy_variable" "excluded_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "Excluded"
  value    = "PlainText"
}

resource "octopusdeploy_variable" "named_excluded_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "NamedExcluded"
  value    = "PlainText"
}

resource "octopusdeploy_variable" "string_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "Test"
  value    = "PlainText"
}

output "octopus_project_1" {
  value = octopusdeploy_project.project_1.id
}

resource "octopusdeploy_variable" "feed_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "HelmFeed"
  value    = data.octopusdeploy_feeds.helm_feed.feeds[0].id
}

resource "octopusdeploy_variable" "account_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "AwsAccount"
  value    = data.octopusdeploy_accounts.example.accounts[0].id
}

resource "octopusdeploy_variable" "gitcred_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "GitCreds"
  value    = data.octopusdeploy_git_credentials.git.git_credentials[0].id
}

resource "octopusdeploy_variable" "workerpool_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "WorkerPool"
  value    = data.octopusdeploy_worker_pools.workerpool_default.worker_pools[0].id
}

resource "octopusdeploy_variable" "certificate_variable" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "Certificate"
  value    = data.octopusdeploy_certificates.example.certificates[0].id
}

resource "octopusdeploy_project" "project_2" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project 2"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test 2"
  project_group_id                     = data.octopusdeploy_project_groups.project_group.project_groups[0].id
  tenanted_deployment_participation    = "Untenanted"
  included_library_variable_sets       = []
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
}

resource "octopusdeploy_process" "test" {
  project_id = "${octopusdeploy_project.project_2.id}"
  depends_on = []
}
resource "octopusdeploy_process_steps_order" "test" {
  process_id = "${octopusdeploy_process.test.id}"
  steps      = ["${octopusdeploy_process_step.process_step_get_mysql_host.id}"]
}

resource "octopusdeploy_process_step" "process_step_get_mysql_host" {
  name                  = "Get MySQL Host"
  type                  = "Octopus.KubernetesRunScript"
  process_id            = "${octopusdeploy_process.test.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  slug                  = "get-mysql-host"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptSource" = "Inline"
  }
  properties            = {
    "Octopus.Action.TargetRoles" = "eks"
  }
}

resource "octopusdeploy_runbook" "runbook" {
  project_id         = octopusdeploy_project.project_1.id
  name               = "MyRunbook"
  description        = "Test Runbook"
  multi_tenancy_mode = "Untenanted"
  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
  retention_policy {
    quantity_to_keep = 10
  }
  environment_scope           = "Specified"
  environments                = []
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_runbook" "runbook2" {
  project_id         = octopusdeploy_project.project_1.id
  name               = "MyRunbook2"
  description        = "Test Runbook"
  multi_tenancy_mode = "Untenanted"
  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
  retention_policy {
    quantity_to_keep = 10
  }
  environment_scope           = "Specified"
  environments                = []
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_runbook" "runbook3" {
  project_id         = octopusdeploy_project.project_1.id
  name               = "MyRunbook3"
  description        = "Test Runbook"
  multi_tenancy_mode = "Untenanted"
  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
  retention_policy {
    quantity_to_keep = 10
  }
  environment_scope           = "Specified"
  environments                = []
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_runbook_process" "runbook" {
  runbook_id = octopusdeploy_runbook.runbook3.id

  step {
    condition           = "Success"
    name                = "Hello world (using PowerShell)"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.Script"
      name                               = "Hello world (using PowerShell)"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = false
      is_required                        = true
      worker_pool_id                     = ""
      properties                         = {
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.ScriptBody"   = "Write-Host 'Hello world, using PowerShell'\n\n#TODO: Experiment with steps of your own :)\n\nWrite-Host '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
        "Octopus.Action.Script.Syntax"       = "PowerShell"
      }
      environments          = []
      excluded_environments = []
      channels              = []
      tenant_tags           = []
      features              = []

      package {
        name                      = "package1"
        package_id                = "package1"
        acquisition_location      = "Server"
        extract_during_deployment = false
        feed_id                   = "#{HelmFeed}"
        properties                = { Extract = "True", Purpose = "", SelectionMode = "immediate" }
      }
    }

    properties   = {}
    target_roles = []
  }

  step {
    condition           = "Success"
    name                = "Test"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.TerraformApply"
      name                               = "Test"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = false
      is_required                        = false
      worker_pool_id                     = ""
      worker_pool_variable               = ""
      properties                         = {
        "Octopus.Action.Terraform.PlanJsonOutput"               = "False"
        "Octopus.Action.Terraform.TemplateDirectory"            = "blah"
        "Octopus.Action.Terraform.ManagedAccount"               = "None"
        "Octopus.Action.Terraform.GoogleCloudAccount"           = "False"
        "Octopus.Action.Script.ScriptSource"                    = "Package"
        "Octopus.Action.GoogleCloud.UseVMServiceAccount"        = "True"
        "Octopus.Action.Terraform.AzureAccount"                 = "False"
        "Octopus.Action.Package.DownloadOnTentacle"             = "False"
        "Octopus.Action.Terraform.RunAutomaticFileSubstitution" = "True"
        "Octopus.Action.GoogleCloud.ImpersonateServiceAccount"  = "False"
        "Octopus.Action.Terraform.AllowPluginDownloads"         = "True"
      }
      environments          = []
      excluded_environments = []
      channels              = []
      tenant_tags           = []
      features              = []

      primary_package {
        package_id           = "terraform"
        acquisition_location = "Server"
        feed_id              = data.octopusdeploy_feeds.built_in_feed.feeds[0].id
        properties           = { SelectionMode = "immediate" }
      }
    }

    properties   = {}
    target_roles = []
  }

  step {
    condition           = "Success"
    name                = "Run an Azure Script"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.AzurePowerShell"
      name                               = "Run an Azure Script"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = false
      is_required                        = false
      worker_pool_id                     = ""
      worker_pool_variable               = ""
      properties                         = {
        "Octopus.Action.Script.Syntax"       = "PowerShell"
        "Octopus.Action.Azure.AccountId"     = data.octopusdeploy_accounts.azure.accounts[0].id
        "Octopus.Action.Script.ScriptBody"   = "echo \"hi\""
        "OctopusUseBundledTooling"           = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
      }
      environments          = []
      excluded_environments = []
      channels              = []
      tenant_tags           = []
      features              = []
      container {
        feed_id = data.octopusdeploy_feeds.docker_feed.feeds[0].id
        image   = "octopusdeploy/worker-tools:6.0.0-ubuntu.22.04"
      }
    }

    properties   = {}
    target_roles = []
  }

  step {
    condition           = "Success"
    name                = "Deploy a Release"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.DeployRelease"
      name                               = "Deploy a Release"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = true
      is_required                        = false
      worker_pool_id                     = data.octopusdeploy_worker_pools.worker_pool_docker.worker_pools[0].id
      worker_pool_variable               = ""
      properties                         = {
        "Octopus.Action.DeployRelease.DeploymentCondition" = "Always"
        "Octopus.Action.DeployRelease.ProjectId" = data.octopusdeploy_projects.other.projects[0].id
      }
      environments                       = []
      excluded_environments              = []
      channels                           = []
      tenant_tags                        = []

      primary_package {
        package_id           = data.octopusdeploy_projects.other.projects[0].id
        acquisition_location = "NotAcquired"
        feed_id              = data.octopusdeploy_feeds.project_feed.feeds[0].id
        properties           = {}
      }

      features = []
    }

    properties   = {}
    target_roles = []
  }
}