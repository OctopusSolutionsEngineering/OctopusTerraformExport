data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
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
  is_version_controlled                = true
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test"
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
  space_id                             = var.octopus_space_id
  included_library_variable_sets       = [octopusdeploy_library_variable_set.octopus_library_variable_set.id]
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }

  git_library_persistence_settings {
    git_credential_id  = "${octopusdeploy_git_credential.gitcredential_matt.id}"
    url                = "https://github.com/mcasperson/octogittest.git"
    base_path          = ".octopus/integrationtest"
    default_branch     = "main"
    protected_branches = ["test"]
  }
}

resource "octopusdeploy_variable" "string_variable" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test"
  value     = "PlainText"
}

resource "octopusdeploy_variable" "string_variable2" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test2"
  value     = "PlainText"
  scope {
    environments = [octopusdeploy_environment.development_environment.id]
  }
}

resource "octopusdeploy_variable" "scoped_var" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "tenantscoped"
  value    = "test"
  scope {
    tenant_tags = ["tag1/a"]
  }
}

resource "octopusdeploy_variable" "string_variable3" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test3"
  value     = "PlainText"
  scope {
    environments = [octopusdeploy_environment.test_environment.id]
  }
}

resource "octopusdeploy_variable" "string_variable4" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "HelmFeed"
  value     = octopusdeploy_helm_feed.feed_helm.id
}

resource "octopusdeploy_variable" "string_variable5" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "UsernamePassword"
  value     = octopusdeploy_username_password_account.account_gke.id
}

resource "octopusdeploy_variable" "string_variable6" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "WorkerPool"
  value     = octopusdeploy_static_worker_pool.workerpool_docker.id
}

resource "octopusdeploy_variable" "string_variable7" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Certificate"
  value     = octopusdeploy_certificate.certificate_kind_ca.id
}

resource "octopusdeploy_variable" "string_variable8" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "TestNull"
  value     = null
}

output "octopus_project_1" {
  value = octopusdeploy_project.project_1.id
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
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
  space_id                             = var.octopus_space_id
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

resource "octopusdeploy_deployment_process" "deployment_process_hello_world" {
  project_id = "${octopusdeploy_project.project_2.id}"

  step {
    condition           = "Success"
    name                = "Hello world (using Bash)"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.Script"
      name                               = "Hello world (using Bash)"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = false
      is_required                        = true
      worker_pool_id                     = octopusdeploy_static_worker_pool.workerpool_docker.id
      properties                         = {
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.ScriptBody" = "echo 'Hello world, using Bash'\n\n#TODO: Experiment with steps of your own :)\n\necho '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
        "Octopus.Action.Script.Syntax" = "Bash"
        "Octopus.Action.RunOnServer" = "true"
      }
      environments                       = []
      excluded_environments              = []
      channels                           = []
      tenant_tags                        = []
      features                           = []
    }

    properties   = {}
    target_roles = []
  }
  depends_on = []
}