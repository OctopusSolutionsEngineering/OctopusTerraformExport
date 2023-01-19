# Octopus Terraform Exporter

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy).

## Downloads

Get the compiled binaries from the [releases](https://github.com/mcasperson/OctopusTerraformExport/releases).

## Usage

To export a complete space, use the following command:

```
./octoterra -url https://yourinstance.octopus.app -space Spaces-## -apiKey API-APIKEYGOESHERE
```

To export a single project and it's associated dependencies, use the following command:

```
./octoterra -url https://yourinstance.octopus.app -space Spaces-## -apiKey API-APIKEYGOESHERE -projectId Projects-1234
```

## To Do

The following resources are not completely exported:
* octopusdeploy_project
  * git config is not exported

The following resources have yet to be exported:
* octopusdeploy_scoped_user_role
* octopusdeploy_team
* octopusdeploy_user
* octopusdeploy_user_role