package terraform

import "time"

type TerraformMachinePolicy struct {
	Type                         string         `hcl:"type,label"`
	Name                         string         `hcl:"name,label"`
	Count                        *string        `hcl:"count"`
	ResourceName                 string         `hcl:"name"`
	Id                           *string        `hcl:"id"`
	SpaceId                      *string        `hcl:"space_id"`
	Description                  *string        `hcl:"description"`
	ConnectionConnectTimeout     *time.Duration `hcl:"connection_connect_timeout"`
	ConnectionRetryCountLimit    *int           `hcl:"connection_retry_count_limit"`
	ConnectionRetrySleepInterval *time.Duration `hcl:"connection_retry_sleep_interval"`
	ConnectionRetryTimeLimit     *time.Duration `hcl:"connection_retry_time_limit"`
	//PollingRequestMaximumMessageProcessingTimeout *time.Duration                     `hcl:"polling_request_maximum_message_processing_timeout"`
	pollingRequestQueueTimeout         *time.Duration                     `hcl:"polling_request_queue_timeout"`
	MachineCleanupPolicy               TerraformMachineCleanupPolicy      `hcl:"machine_cleanup_policy,block"`
	TerraformMachineConnectivityPolicy TerraformMachineConnectivityPolicy `hcl:"machine_connectivity_policy,block"`
	TerraformMachineHealthCheckPolicy  TerraformMachineHealthCheckPolicy  `hcl:"machine_health_check_policy,block"`
	TerraformMachineUpdatePolicy       TerraformMachineUpdatePolicy       `hcl:"machine_update_policy,block"`
}

type TerraformMachineCleanupPolicy struct {
	DeleteMachinesBehavior        *string        `hcl:"delete_machines_behavior"`
	DeleteMachinesElapsedTimespan *time.Duration `hcl:"delete_machines_elapsed_timespan"`
}

type TerraformMachineConnectivityPolicy struct {
	MachineConnectivityBehavior string `hcl:"machine_connectivity_behavior"`
}

type TerraformMachineHealthCheckPolicy struct {
	BashHealthCheckPolicy       TerraformBashHealthCheckPolicy       `hcl:"bash_health_check_policy,block"`
	PowershellHealthCheckPolicy TerraformPowershellHealthCheckPolicy `hcl:"powershell_health_check_policy,block"`
	HealthCheckCron             *string                              `hcl:"health_check_cron"`
	HealthCheckCronTimezone     *string                              `hcl:"health_check_cron_timezone"`
	HealthCheckInterval         *time.Duration                       `hcl:"health_check_interval"`
	HealthCheckType             *string                              `hcl:"health_check_type"`
}

type TerraformBashHealthCheckPolicy struct {
	RunType    string `hcl:"run_type"`
	ScriptBody string `hcl:"script_body"`
}

type TerraformPowershellHealthCheckPolicy struct {
	RunType    string `hcl:"run_type"`
	ScriptBody string `hcl:"script_body"`
}

type TerraformMachineUpdatePolicy struct {
	CalamariUpdateBehavior        *string `hcl:"calamari_update_behavior"`
	TentacleUpdateAccountId       *string `hcl:"tentacle_update_account_id"`
	TentacleUpdateBehavior        *string `hcl:"tentacle_update_behavior"`
	KubernetesAgentUpdateBehavior *string `hcl:"kubernetes_agent_update_behavior"`
}
