resource "octopusdeploy_tag" "tagset_cities_tag_sydney" {
  name        = "Sydney"
  tag_set_id  = "${octopusdeploy_tag_set.tagset_cities.id}"
  color       = "#333333"
  description = ""
}
