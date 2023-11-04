data "octopusdeploy_feeds" "docker_feed" {
  feed_type    = "Docker"
  ids          = null
  partial_name = "Docker"
  skip         = 0
  take         = 1
}