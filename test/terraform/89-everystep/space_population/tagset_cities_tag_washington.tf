resource "octopusdeploy_tag" "tagset_cities_tag_washington" {
  name        = "Washington"
  tag_set_id  = "${octopusdeploy_tag_set.tagset_cities.id}"
  color       = "#5ECD9E"
  description = ""
}
