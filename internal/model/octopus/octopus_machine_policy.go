package octopus

type MachinePolicy struct {
	Id                                            string
	Name                                          string
	SpaceId                                       *string
	Description                                   *string
	IsDefault                                     bool
	PollingRequestQueueTimeout                    string
	PollingRequestMaximumMessageProcessingTimeout string
	ConnectionRetrySleepInterval                  string
	ConnectionRetryCountLimit                     *int
	ConnectionRetryTimeLimit                      string
	ConnectionConnectTimeout                      string
	MachineHealthCheckPolicy                      MachineHealthCheckPolicy
	MachineConnectivityPolicy                     MachineConnectivityPolicy
	MachineCleanupPolicy                          MachineCleanupPolicy
	MachineUpdatePolicy                           MachineUpdatePolicy
}

type MachineHealthCheckPolicy struct {
	HealthCheckInterval         *string
	HealthCheckCron             *string
	HealthCheckCronTimezone     *string
	HealthCheckType             *string
	PowerShellHealthCheckPolicy PowerShellHealthCheckPolicy
	BashHealthCheckPolicy       BashHealthCheckPolicy
}

type PowerShellHealthCheckPolicy struct {
	RunType    string
	ScriptBody string
}

type BashHealthCheckPolicy struct {
	RunType    string
	ScriptBody string
}

type MachineConnectivityPolicy struct {
	MachineConnectivityBehavior string
}

type MachineCleanupPolicy struct {
	DeleteMachinesBehavior        string
	DeleteMachinesElapsedTimeSpan string
}

type MachineUpdatePolicy struct {
	CalamariUpdateBehavior  *string
	TentacleUpdateBehavior  *string
	TentacleUpdateAccountId *string
}
