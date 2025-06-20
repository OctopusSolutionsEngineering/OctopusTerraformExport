variable "project_every_step_project_step_deploy_kubernetes_yaml_from_a_package_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Deploy Kubernetes YAML from a package in project Every Step Project"
  default     = "K8sApplication"
}
resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_kubernetes_yaml_from_a_package" {
  name                  = "Deploy Kubernetes YAML from a package"
  type                  = "Octopus.KubernetesDeployRawYaml"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step provides an example deploying a Kubernetes YAML file from a package."
  package_requirement   = "LetOctopusDecide"
  primary_package       = { acquisition_location = "Server", feed_id = "${data.octopusdeploy_feeds.feed_octopus_server__built_in_.feeds[0].id}", id = null, package_id = "${var.project_every_step_project_step_deploy_kubernetes_yaml_from_a_package_packageid}", properties = { SelectionMode = "immediate" } }
  slug                  = "deploy-kubernetes-yaml-from-a-package"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "Kubernetes"
      }
  execution_properties  = {
        "Octopus.Action.Kubernetes.ResourceStatusCheck" = "True"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Kubernetes.ServerSideApply.Enabled" = "True"
        "Octopus.Action.KubernetesContainers.CustomResourceYaml" = "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\n  labels:\n    app: nginx\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: nginx\n  template:\n    metadata:\n      labels:\n        app: nginx\n    spec:\n      containers:\n      - name: nginx\n        image: nginx:1.14.2\n        ports:\n        - containerPort: 80"
        "Octopus.Action.Kubernetes.DeploymentTimeout" = "180"
        "Octopus.Action.RunOnServer" = "true"
        "Octopus.Action.Script.ScriptSource" = "Package"
        "Octopus.Action.Kubernetes.ServerSideApply.ForceConflicts" = "True"
        "Octopus.Action.KubernetesContainers.CustomResourceYamlFileName" = "deployment.yaml"
      }
}
