data "octopusdeploy_accounts" "example" {
  account_type = "AmazonWebServicesAccount"
  ids          = []
  partial_name = "AWS Account"
  skip         = 0
  take         = 1
}