resource "octopusdeploy_tag" "tagset_cities_tag_london" {
  name        = "London"
  tag_set_id  = "${octopusdeploy_tag_set.tagset_cities.id}"
  color       = "#87BFEC"
  description = ""
}
