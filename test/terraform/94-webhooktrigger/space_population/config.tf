terraform {
  required_providers {
    # Webhook triggers were added to the provider in 1.19.3
    octopusdeploy = { source = "OctopusDeploy/octopusdeploy", version = "1.19.3" }
  }
}
