/*
  These resources are all scoped to a parent environment rather than a regular environment.
  They verify that the exported module resolves parent environments in the same places that
  regular environments are resolved.
*/

resource "octopusdeploy_username_password_account" "account_parent_env" {
  description                       = "An account scoped to a parent environment"
  name                              = "Parent Env Account"
  environments                      = [octopusdeploy_parent_environment.example.id]
  tenant_tags                       = []
  tenants                           = null
  tenanted_deployment_participation = "Untenanted"
  username                          = "admin"
  password                          = "secretgoeshere"
}

resource "octopusdeploy_certificate" "certificate_parent_env" {
  name                              = "Parent Env Certificate"
  certificate_data                  = file("dummycert.txt")
  password                          = "Password01!"
  environments                      = [octopusdeploy_parent_environment.example.id]
  notes                             = "A certificate scoped to a parent environment"
  tenant_tags                       = []
  tenanted_deployment_participation = "Untenanted"
  tenants                           = []
}

/*
  Note that Octopus rejects parent environments in some places that accept regular environments,
  so they are not covered here:
    * A lifecycle phase fails with "The phase ... has an invalid environment ID".
    * Linking a tenant to a project fails with "You are attempting to add environments that you
      do not have permission to view".
*/

resource "octopusdeploy_variable" "parent_env_variable" {
  name         = "Test.ParentEnvVariable"
  type         = "String"
  description  = "A variable scoped to a parent environment"
  is_sensitive = false
  owner_id     = octopusdeploy_project.deploy_frontend_project.id
  value        = "parent"

  scope {
    environments = [octopusdeploy_parent_environment.example.id]
  }
}

resource "octopusdeploy_project_scheduled_trigger" "parent_env_runbook_trigger" {
  name        = "Parent Env Runbook Trigger"
  description = "A trigger that runs a runbook against a parent environment"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id    = var.octopus_space_id

  run_runbook_action {
    target_environment_ids = [octopusdeploy_parent_environment.example.id]
    runbook_id             = octopusdeploy_runbook.runbook.id
  }

  cron_expression_schedule {
    cron_expression = "0 0 06 * * Mon-Fri"
  }
}
