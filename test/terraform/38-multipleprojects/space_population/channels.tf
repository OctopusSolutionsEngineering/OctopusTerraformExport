resource "octopusdeploy_channel" "channel1" {
  name        = "Test 1"
  project_id  = octopusdeploy_project.project_1.id
  description = "Test channel"
  is_default  = true
}

resource "octopusdeploy_channel" "channel2" {
  name        = "Test 2"
  project_id  = octopusdeploy_project.project_2.id
  description = "Test channel"
  is_default  = true
}