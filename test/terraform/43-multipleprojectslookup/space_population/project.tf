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

  git_library_persistence_settings {
    git_credential_id  = data.octopusdeploy_git_credentials.git.git_credentials[0].id
    url                = "https://github.com/mcasperson/octogittest.git"
    base_path          = ".octopus/integrationtest"
    default_branch     = "main"
    protected_branches = []
  }
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
  force_package_download      = true
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
  force_package_download      = true
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
  force_package_download      = true
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
        feed_id                   = data.octopusdeploy_feeds.built_in_feed.feeds[0].id
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
}