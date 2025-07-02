resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_with_kustomize" {
  name                  = "Deploy with Kustomize"
  type                  = "Octopus.Kubernetes.Kustomize"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  git_dependencies      = { "" = { default_branch = "main", file_path_filters = null, git_credential_id = "${octopusdeploy_git_credential.gitcredential_github.id}", git_credential_type = "Library", repository_uri = "https://github.com/OctopusSamples/OctoPetShop.git" } }
  notes                 = "This step deploys a Kustomize resource to a Kubernetes cluster."
  package_requirement   = "LetOctopusDecide"
  slug                  = "deploy-with-kustomize"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "Kubernetes"
      }
  execution_properties  = {
        "Octopus.Action.Kubernetes.ServerSideApply.Enabled" = "True"
        "Octopus.Action.Kubernetes.ServerSideApply.ForceConflicts" = "True"
        "Octopus.Action.Kubernetes.Kustomize.OverlayPath" = "overlays/#{Octopus.Environment.Name}"
        "Octopus.Action.SubstituteInFiles.TargetFiles" = " **/*.env"
        "Octopus.Action.Kubernetes.ResourceStatusCheck" = "True"
        "Octopus.Action.Kubernetes.DeploymentTimeout" = "180"
        "Octopus.Action.Script.ScriptSource" = "GitRepository"
        "Octopus.Action.GitRepository.Source" = "External"
      }
}
