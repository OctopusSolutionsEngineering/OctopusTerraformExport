data "octopusdeploy_accounts" "azure" {
  account_type = "AzureServicePrincipal"
  ids          = []
  partial_name = "Azure"
  skip         = 0
  take         = 1
}