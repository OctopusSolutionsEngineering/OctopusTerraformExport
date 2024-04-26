resource "octopusdeploy_project_scheduled_trigger" "once_daily_example" {
  name        = "Once Daily example"
  description = "This is a once daily schedule"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_new_release_action {
    destination_environment_id = octopusdeploy_environment.test_environment.id
  }
  once_daily_schedule {
    start_time   = "2024-03-22T09:00:00"
    days_of_week = ["Tuesday", "Wednesday", "Monday"]
  }
}

resource "octopusdeploy_project_scheduled_trigger" "continuous_example" {
  name        = "Continuous"
  description = "This is a continuous daily schedule"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_new_release_action {
    destination_environment_id = octopusdeploy_environment.test_environment.id
  }
  continuous_daily_schedule {
    interval      = "OnceHourly"
    hour_interval = 3
    run_after     = "2024-03-22T09:00:00"
    run_until     = "2024-03-29T13:00:00"
    days_of_week  = ["Monday", "Tuesday", "Friday"]
  }
}

resource "octopusdeploy_project_scheduled_trigger" "deploy_latest_example" {
  name       = "Deploy Latest"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_latest_release_action {
    source_environment_id      = octopusdeploy_environment.development_environment.id
    destination_environment_id = octopusdeploy_environment.test_environment.id
    should_redeploy            = true
  }
  cron_expression_schedule {
    cron_expression = "0 0 06 * * Mon-Fri"
  }
}

resource "octopusdeploy_project_scheduled_trigger" "deploy_new_example" {
  name       = "Deploy New"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_new_release_action {
    destination_environment_id = octopusdeploy_environment.test_environment.id
  }
  cron_expression_schedule {
    cron_expression = "0 0 06 * * Mon-Fri"
  }
}

resource "octopusdeploy_project_scheduled_trigger" "date_of_month" {
  name       = "Date Of Month"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_new_release_action {
    destination_environment_id = octopusdeploy_environment.test_environment.id
  }
  days_per_month_schedule {
    monthly_schedule_type  = "DateOfMonth"
    start_time = "2024-03-22T09:00:00"
    date_of_month = "1"
    day_number_of_month = "1"
    day_of_week = "Monday"
  }
}

resource "octopusdeploy_project_scheduled_trigger" "day_of_month" {
  name       = "Day Of Month"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  deploy_new_release_action {
    destination_environment_id = octopusdeploy_environment.test_environment.id
  }
  days_per_month_schedule {
    monthly_schedule_type  = "DayOfMonth"
    start_time = "2024-03-22T09:00:00"
    date_of_month = "1"
    day_number_of_month = "1"
    day_of_week = "Monday"
  }
}

resource "octopusdeploy_project_scheduled_trigger" "runbook_example" {
  name        = "Runbook"
  description = "This is a Cron schedule"
  project_id  = octopusdeploy_project.deploy_frontend_project.id
  space_id = var.octopus_space_id
  run_runbook_action {
    target_environment_ids = [octopusdeploy_environment.test_environment.id, octopusdeploy_environment.development_environment.id]
    runbook_id             = octopusdeploy_runbook.runbook.id
  }
  cron_expression_schedule {
    cron_expression = "0 0 06 * * Mon-Fri"
  }
}