resource "octopusdeploy_process_step" "process_step_every_step_project_deploy_kubernetes_yaml" {
  name                  = "Deploy Kubernetes YAML"
  type                  = "Octopus.KubernetesDeployRawYaml"
  process_id            = "${octopusdeploy_process.process_every_step_project.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  notes                 = "This step deploys a raw YAML document to a Kubernetes cluster."
  package_requirement   = "LetOctopusDecide"
  slug                  = "deploy-kubernetes-yaml"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
        "Octopus.Action.TargetRoles" = "Kubernetes"
      }
  execution_properties  = {
        "Octopus.Action.KubernetesContainers.CustomResourceYaml" = "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\n  labels:\n    app: nginx\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: nginx\n  template:\n    metadata:\n      labels:\n        app: nginx\n    spec:\n      containers:\n      - name: nginx\n        image: nginx:1.14.2\n        ports:\n        - containerPort: 80"
        "Octopus.Action.RunOnServer" = "true"
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Kubernetes.ResourceStatusCheck" = "True"
        "Octopus.Action.Kubernetes.DeploymentTimeout" = "180"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Kubernetes.ServerSideApply.Enabled" = "True"
        "Octopus.Action.Kubernetes.ServerSideApply.ForceConflicts" = "True"
      }
}
