# Octopus Terraform Exporter

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy).

## Downloads

Get the compiled binaries from the [releases](https://github.com/mcasperson/OctopusTerraformExport/releases).

## Support

Feel free to raise an [issue](https://github.com/OctopusSolutionsEngineering/OctopusTerraformExport/issues) with any bugs of feature enhancements. 

Note this open source project has no offical support.

## Usage

To export a complete space, use the following command:

```
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -dest /tmp/octoexport
```

To export a single project and it's associated dependencies, use the following command:

```
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectId Projects-1234 \
    -dest /tmp/octoexport
```

## Browser usage

Exporting projects to HCL can be embedded in the browser by using the [Violentmonkey](https://violentmonkey.github.io/)
script [violentmonkey.js](wasm/violentmonkey.js).

![image](https://user-images.githubusercontent.com/160104/227693138-3fd77272-d962-444b-a50f-735174629711.png)

This script adds a `Export HCL` link to the project page. Once the link is ready to be clicked (it takes a minute or
so to build the HCL), the link displays the project's HCL representation in a popup window.

![HCL Export link](hcl_export.png)

## To Do

The following resources have yet to be exported:
* octopusdeploy_scoped_user_role
* octopusdeploy_team
* octopusdeploy_user
* octopusdeploy_user_role

## Report Card
![Go Report Card](https://goreportcard.com/badge/mcasperson/OctopusTerraformExport)
