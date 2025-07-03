package terraform

type TerraformStepWithWorkerPool interface {
	SetWorkerPoolId(workerPoolId string)
}
