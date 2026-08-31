# A webhook trigger authenticated with a shared secret. The API never returns the secret value, so the
# exported module exposes it as a Terraform variable.
resource "octopusdeploy_webhook_trigger" "webhook_secret_example" {
  name        = "Webhook Secret"
  description = "This is a webhook trigger authenticated with a shared secret"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id    = var.octopus_space_id
  secret      = "aVerySecretSecret"

  run_runbook_action = {
    runbook_id             = octopusdeploy_runbook.runbook.id
    target_environment_ids = [
      octopusdeploy_environment.development_environment.id,
      octopusdeploy_environment.test_environment.id
    ]
  }
}

# A disabled webhook trigger authenticated with an Octopus API key. These triggers have no secret at all,
# so the exported module must not define a secret variable for them.
resource "octopusdeploy_webhook_trigger" "webhook_api_key_example" {
  name            = "Webhook Api Key"
  description     = "This is a webhook trigger authenticated with an API key"
  project_id      = octopusdeploy_project.deploy_frontend_project.id
  space_id        = var.octopus_space_id
  is_disabled     = true
  require_api_key = true

  run_runbook_action = {
    runbook_id             = octopusdeploy_runbook.runbook.id
    target_environment_ids = [octopusdeploy_environment.development_environment.id]
  }
}
