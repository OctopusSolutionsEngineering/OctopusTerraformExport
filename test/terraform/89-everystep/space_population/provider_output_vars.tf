output "octopus_server" {
  value = "${var.octopus_server}"
}
output "octopus_space_id" {
  value = "${var.octopus_space_id}"
}
data "octopusdeploy_spaces" "octopus_space_name" {
  ids  = ["${var.octopus_space_id}"]
  skip = 0
  take = 1
}
output "octopus_space_name" {
  value = "${data.octopusdeploy_spaces.octopus_space_name.spaces[0].name}"
}
