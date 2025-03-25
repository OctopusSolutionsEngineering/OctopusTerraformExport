resource "octopusdeploy_git_credential" "gitcredential_matt" {
  name     = "matt"
  type     = "UsernamePassword"
  username = "mcasperson"
  password = var.gitcredential_matt_sensitive_value
}

variable "gitcredential_matt_sensitive_value" {
  type        = string
  nullable    = false
  sensitive   = true
  description = "The secret variable value associated with the git credential \"matt\""
}