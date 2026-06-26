resource "octopusdeploy_lifecycle" "simple_lifecycle" {
  description = "A test lifecycle"
  name        = "Simple"

  release_retention_with_strategy {
    strategy         = "Count"
    quantity_to_keep = 30
    unit             = "Days"
  }

  tentacle_retention_with_strategy {
    strategy         = "Count"
    quantity_to_keep = 30
    unit             = "Days"
  }

  phase {
    automatic_deployment_targets = []
    optional_deployment_targets  = [octopusdeploy_environment.development_environment.id]
    name                         = octopusdeploy_environment.development_environment.name

    release_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }

    tentacle_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }
  }

  phase {
    automatic_deployment_targets = []
    optional_deployment_targets  = [octopusdeploy_environment.test_environment.id]
    name                         = octopusdeploy_environment.test_environment.name

    release_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }

    tentacle_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }
  }

  phase {
    automatic_deployment_targets = []
    optional_deployment_targets  = [octopusdeploy_environment.production_environment.id]
    name                         = octopusdeploy_environment.production_environment.name

    release_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }

    tentacle_retention_with_strategy {
      strategy         = "Count"
      quantity_to_keep = 30
      unit             = "Days"
    }
  }
}