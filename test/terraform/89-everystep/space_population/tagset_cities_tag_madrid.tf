resource "octopusdeploy_tag" "tagset_cities_tag_madrid" {
  name        = "Madrid"
  tag_set_id  = "${octopusdeploy_tag_set.tagset_cities.id}"
  color       = "#FFA461"
  description = ""
}
