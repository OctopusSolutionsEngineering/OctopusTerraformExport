resource "octopusdeploy_docker_container_registry" "feed_docker" {
  name                                 = "Docker"
  api_version                          = "v1"
  feed_uri                             = "https://index.docker.io"
  package_acquisition_location_options = ["ExecutionTarget", "NotAcquired"]
}
