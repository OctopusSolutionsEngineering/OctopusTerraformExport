resource "octopusdeploy_s3_feed" "s3" {
  name = "S3"
  use_machine_credentials = false
  access_key = "given_access_key"
  secret_key = "some_secret_key"
}