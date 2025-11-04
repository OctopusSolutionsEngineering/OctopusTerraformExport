# Octopus Terraform Exporter
![Coverage](https://img.shields.io/badge/Coverage-69.7%25-yellow)

[![Github All Releases](https://img.shields.io/github/downloads/OctopusSolutionsEngineering/OctopusTerraformExport/total.svg)]()

This app exports an Octopus space to the associated Terraform resources for use with the 
[Octopus Terraform Provider](https://registry.terraform.io/providers/OctopusDeploy/octopusdeploy/latest).

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

Individual runbooks can be exported using data lookups to reference their parent project (i.e. when exporting a runbook,
`-lookupProjectDependencies` is implied).

Note that when exporting an individual runbook, the project variables are not exported. These runbooks must be applied to
projects with any required project variables or library variable sets already defined.

The `-runbookName` argument requires either `-projectName` or `-projectId` because runbooks do not have a unique name:

```bash
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectId Projects-1234 \
    -runbookName "Backup Database"
    -dest /tmp/octoexport
```

The `-runbookId` argument does not need a project to be defined:

```bash
./octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -runbookId "Runbooks-123"
    -dest /tmp/octoexport
```

Docker can also be used to run Octoterra:

```bash
docker run -v $PWD:/tmp/octoexport --rm ghcr.io/octopussolutionsengineering/octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectName YourProject \
    -lookupProjectDependencies \
    -dest /tmp/octoexport
```

## Creating reference architecture step templates

The `Apply a Terraform template` step in Octopus can be used to execute the Terraform modules created with `octoterra`.
One limitation of this step is that it does not persist the local state, so you must use remote state for most
scenarios.

However, the fact that local state is not maintained between step executions can be used to build "stateless" modules.
Stateless modules use a `resource` and `data` block pair to look for existing resources and only create a new resource
if it does not already exist. For example, the following code creates an environment only if it doesn't already exist:

```hcl
data "octopusdeploy_environments" "environment_development" {
  ids          = null
  partial_name = "Development"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_environment" "environment_development" {
  count                        = "${length(data.octopusdeploy_environments.environment_development.environments) != 0 ? 0 : 1}"
  name                         = "Development"
  description                  = "A test environment"
  allow_dynamic_infrastructure = true
  use_guided_failure           = false
  sort_order                   = 0

  jira_extension_settings {
    environment_type = "unmapped"
  }

  jira_service_management_extension_settings {
    is_enabled = false
  }

  servicenow_extension_settings {
    is_enabled = false
  }
  lifecycle {
    prevent_destroy = true
  }
}
```

The ID of the environment can be then be accessed with:

```hcl
${length(data.octopusdeploy_environments.environment_development.environments) != 0 ? data.octopusdeploy_environments.environment_development.environments[0].id : octopusdeploy_environment.environment_development[0].id}
```

When used with a persistent state, the example above would first create the environment and then delete it when
the module was applied for a second time. However, when the state is not maintained between calls to `terraform apply`,
the example above either creates an environment if it doesn't exist, or references the existing environment.

We exploit this behaviour to create "stateless" modules that are executed by a template `Apply a Terraform template` step.
The command below uses the `-stepTemplate`, `-stepTemplateName`, and `-stepTemplateKey` arguments to generate a step
template JSON file that captures the Octopus resources in the space passed to `-space` as a stateless module:

```bash
docker run -v $PWD:/tmp/octoexport --rm ghcr.io/octopussolutionsengineering/octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-123 \
    -apiKey API-ABCDEFGHIJKLMNOPQURTUVWXYZ \
    -stepTemplate \
    -stepTemplateName "My Reference Architecture" \
    -stepTemplateKey "Kubernetes" \ 
    -dest /tmp/octoexport
```

This command produces a file called `step-template.json` that can be imported as an Octopus step template. It exposes
all the secret variables required to deploy the module as step template parameters, allowing users to apply (and reapply)
the template into any space, reusing any existing Octopus resources and creating any that are missing. This is a convenient
method for composing Octopus resources in a space without having to worry about configuring any external persistent
state.

## Octopus integration

The documentation in [platform engineering](https://octopus.com/docs/platform-engineering) allow octoterra to be used directly in Octopus via native steps.

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

### Reporting issues

You can report issues [here](https://github.com/OctopusSolutionsEngineering/OctopusTerraformExport/issues).

## Tool Pack Documentation

The Tool Pack documentation is found [here](https://docs.google.com/document/d/18CeeWZ_olJEy-87PIxFx7x2lhPWHTiYaQXvPBDxQFGA/edit) (internal access only).

## Troubleshooting

If you get an error like this when using the Docker image:

```bash
error	mkdir /tmp/octoexport/space_creation: permission denied
```

You may need to use the `z` option to modify the selinux label. 

WARNING! Using the `z` option can have some serious unintended side effects. See the post at https://stackoverflow.com/a/35222815/157605 for more details.

This is an example using the `z` option:

```bash
docker run -v $PWD:/tmp/octoexport:z --rm ghcr.io/octopussolutionsengineering/octoterra \
    -url https://yourinstance.octopus.app \
    -space Spaces-## \
    -apiKey API-APIKEYGOESHERE \
    -projectName YourProject \
    -lookupProjectDependencies \
    -dest /tmp/octoexport
```