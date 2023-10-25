resource "octopusdeploy_tag_set" "tagset_tag1" {
  name        = "type"
  description = "Test tagset"
  sort_order  = 0
}

resource "octopusdeploy_tag" "tag_a" {
  name        = "a"
  color       = "#333333"
  description = "tag a"
  sort_order  = 2
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}

resource "octopusdeploy_tag" "tag_b" {
  name        = "b"
  color       = "#333333"
  description = "tag b"
  sort_order  = 3
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}

resource "octopusdeploy_tag" "tag_excluded" {
  name        = "excluded"
  color       = "#333333"
  description = "excluded"
  sort_order  = 4
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}

resource "octopusdeploy_tag" "tag_ignore" {
  name        = "ignorethis"
  color       = "#333333"
  description = "ignore this"
  sort_order  = 5
  tag_set_id = octopusdeploy_tag_set.tagset_tag1.id
}