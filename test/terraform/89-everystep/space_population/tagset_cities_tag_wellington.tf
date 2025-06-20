resource "octopusdeploy_tag" "tagset_cities_tag_wellington" {
  name        = "Wellington"
  tag_set_id  = "${octopusdeploy_tag_set.tagset_cities.id}"
  color       = "#C5AEEE"
  description = ""
}
