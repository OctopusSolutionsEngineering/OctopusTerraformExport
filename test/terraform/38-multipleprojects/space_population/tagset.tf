resource "octopusdeploy_tag_set" "tagset_tag1" {
  name        = "tag1"
  description = "Test tagset"
}

resource "octopusdeploy_tag" "tag_a" {
  name        = "a"
  color       = "#333333"
  description = "tag a"
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}

resource "octopusdeploy_tag" "tag_b" {
  name        = "b"
  color       = "#333333"
  description = "tag b"
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}

resource "octopusdeploy_tag_set" "tagset_tag2" {
  name        = "tag2"
  description = "Test tagset"
}

resource "octopusdeploy_tag" "tag_c" {
  name        = "c"
  color       = "#333333"
  description = "tag c"
  tag_set_id = octopusdeploy_tag_set.tagset_tag2.id
}

resource "octopusdeploy_tag" "tag_d" {
  name        = "d"
  color       = "#333333"
  description = "tag d"
  tag_set_id = octopusdeploy_tag_set.tagset_tag2.id
}