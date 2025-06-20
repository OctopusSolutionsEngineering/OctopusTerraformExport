resource "octopusdeploy_docker_container_registry" "feed_github_container_registry" {
  name                                 = "GitHub Container Registry"
  registry_path                        = ""
  api_version                          = "v2"
  feed_uri                             = "https://ghcr.io"
  package_acquisition_location_options = ["ExecutionTarget", "NotAcquired"]
  lifecycle {
    ignore_changes = [password]
  }
}
