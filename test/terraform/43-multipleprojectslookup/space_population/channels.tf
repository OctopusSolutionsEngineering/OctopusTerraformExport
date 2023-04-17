data "octopusdeploy_lifecycles" "lifecycle" {
  ids          = []
  partial_name = "Simple"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_channel" "channel1" {
  name        = "Test 1"
  project_id  = octopusdeploy_project.project_1.id
  description = "Test channel"
  is_default  = true
  lifecycle_id = data.octopusdeploy_lifecycles.lifecycle.lifecycles[0].id
}

resource "octopusdeploy_channel" "channel2" {
  name        = "Test 2"
  project_id  = octopusdeploy_project.project_2.id
  description = "Test channel"
  is_default  = true
}