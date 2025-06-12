resource "octopusdeploy_machine_proxy" "machineproxy" {
  name        = "Test"
  host = "localhost"
  username = "admin"
  password = "password"
  port = 100
}
