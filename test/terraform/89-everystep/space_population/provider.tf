provider "octopusdeploy" {
  address  = "${trimspace(var.octopus_server)}"
  api_key  = "${trimspace(var.octopus_apikey)}"
  space_id = "${trimspace(var.octopus_space_id)}"
}
