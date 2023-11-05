# Octopus Terraform Exporter
![Coverage](https://img.shields.io/badge/Coverage-83.3%25-brightgreen)

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy).

## Downloads

Get the compiled binaries from the [releases](https://github.com/OctopusSolutionsEngineering/OctopusTerraformExport/releases).

## Usage

To export a complete space, use the following command:

```bash
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -dest /tmp/octoexport
```

To export a single project and it's associated dependencies, use the following command:

```bash
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectId Projects-1234 \
    -dest /tmp/octoexport
```

Projects can also be exported using data source lookups to reference existing external resources rather than creating them. 
This is useful when exporting a project to be reimported into a space where all the existing resources like environments, accounts,
feeds, git credentials, targets, and worker pools already exist.

To do so, use the following command:

```bash
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectId Projects-1234 \
    -lookupProjectDependencies \
    -dest /tmp/octoexport
```

Octoterra is also able to be run as a Docker image:

```
docker run -v $PWD:/tmp/octoexport --rm octopussamples/octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectName YourProject \
    -lookupProjectDependencies \
    -dest /tmp/octoexport
```

## Browser usage

Exporting projects to HCL can be embedded in the browser by using the [Violentmonkey](https://violentmonkey.github.io/)
script [violentmonkey.js](wasm/violentmonkey.js).

![image](https://user-images.githubusercontent.com/160104/227693138-3fd77272-d962-444b-a50f-735174629711.png)

This script adds a `Export HCL` link to the project page. Once the link is ready to be clicked (it takes a minute or
so to build the HCL), the link displays the project's HCL representation in a popup window.

![HCL Export link](hcl_export.png)

## How to contribute

* Ensure any new features or bugs have an associated test in [octoterra_test.go](https://github.com/OctopusSolutionsEngineering/OctopusTerraformExport/blob/main/cmd/octoterra_test.go)
* Tests can be run locally by setting the following environment variables:
    * `ECR_ACCESS_KEY` and `ECR_SECRET_KEY` environment variables to an AWS access and secret key. These need to be valid AWS credentials but do not need any particular permissions.
    * `LICENSE` to a base64 encoded Octopus license key.
* Ignore tests that require the `GIT_CREDENTIAL` variable to be set, as these tests require specific credentials to GitHub repositories.

### Octopus engineers

If your feature or bug is regarding CaC enabled projects that require valid Git credentials to be tested, reach out in Slack in #se-tool-requests. Credentials for these tests will need to be added to the GitHub repo.

This project uses continuous integration, so push your changes to `main`.

### External contributors

If your feature or bug is regarding CaC enabled projects that require valid Git credentials to be tested, allow the credentials and the Git repo to be configured via environment variables. Make a note of the environment variables in the pull requests so appropriate values can be defined as secrets in this repo.

Create a pull request against `main`.

## Tool Pack Documentation

The Tool Pack documentation is found [here](https://docs.google.com/document/d/18CeeWZ_olJEy-87PIxFx7x2lhPWHTiYaQXvPBDxQFGA/edit) (internal access only).

## To Do

The following resources have yet to be exported:
* octopusdeploy_scoped_user_role
* octopusdeploy_team
* octopusdeploy_user
* octopusdeploy_user_role

Features:
* Exclude channels
* Exclude triggers
* Ignore tenanted, versioning
