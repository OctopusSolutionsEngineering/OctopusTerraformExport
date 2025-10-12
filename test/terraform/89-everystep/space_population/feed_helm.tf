variable "feed_helm_password" {
  type        = string
  nullable    = false
  sensitive   = true
  description = "The password used by the feed Helm"
  default     = "Change Me!"
}
resource "octopusdeploy_helm_feed" "feed_helm" {
  name                                 = "Helm"
  password                             = "${var.feed_helm_password}"
  feed_uri                             = "https://charts.helm.sh/stable"
  username                             = "username"
  package_acquisition_location_options = ["ExecutionTarget", "Server", "NotAcquired"]
  lifecycle {
    ignore_changes = [password]
  }
}
