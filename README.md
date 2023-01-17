# Octopus Terraform Exporter

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy).

## Downloads

Get the compiled binaries from the [releases](https://github.com/mcasperson/OctopusTerraformExport/releases).

## Usage

```
./octoterra -url https://yourinstance.octopus.app -space Spaces-## -apiKey API-APIKEYGOESHERE
```

## To Do

The following resources are not completely exported:
* octopusdeploy_project

The following resources have yet to be exported:
* octopusdeploy_azure_cloud_service_deployment_target
* octopusdeploy_azure_service_fabric_cluster_deployment_target
* octopusdeploy_azure_web_app_deployment_target
* octopusdeploy_cloud_region_deployment_target
* octopusdeploy_offline_package_drop_deployment_target
* octopusdeploy_polling_tentacle_deployment_target

* octopusdeploy_scoped_user_role
* octopusdeploy_team
* octopusdeploy_user
* octopusdeploy_user_role