package converters

import (
	"fmt"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployDeploymentFreezeDataType = "octopusdeploy_deployment_freezes"
const octopusdeployDeploymentFreezeProjectResourceType = "octopusdeploy_deployment_freeze_project"
const octopusdeployDeploymentFreezeTenantResourceType = "octopusdeploy_deployment_freeze_tenant"
const octopusdeployDeploymentFreezeResourceType = "octopusdeploy_deployment_freeze"

type DeploymentFreezeConverter struct {
	Client                         client.OctopusClient
	ErrGroup                       *errgroup.Group
	ExcludeDeploymentFreezes       args.StringSliceArgs
	ExcludeDeploymentFreezesRegex  args.StringSliceArgs
	ExcludeDeploymentFreezesExcept args.StringSliceArgs
	ExcludeAllDeploymentFreezes    bool
	Excluder                       ExcludeByName
	LimitResourceCount             int
	IncludeIds                     bool
	GenerateImportScripts          bool
}

func (c DeploymentFreezeConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c DeploymentFreezeConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c DeploymentFreezeConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllDeploymentFreezes {
		return nil
	}

	done := make(chan struct{})
	defer close(done)

	freezes := octopus.DeploymentFreezes{}
	if err := c.Client.GetAllGlobalResources("DeploymentFreezes", &freezes, []string{"skip", "0"}, []string{"take", "10000"}); err != nil {
		return err
	}

	for _, resource := range freezes.DeploymentFreezes {

		if c.Excluder.IsResourceExcludedWithRegex(resource.Name,
			c.ExcludeAllDeploymentFreezes,
			c.ExcludeDeploymentFreezes,
			c.ExcludeDeploymentFreezesRegex,
			c.ExcludeDeploymentFreezesExcept) {
			continue
		}

		zap.L().Info("Deployment Freeze: " + resource.Id + " " + resource.Name)
		err := c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c DeploymentFreezeConverter) GetResourceType() string {
	return "DeploymentFreezes"
}

func (c DeploymentFreezeConverter) toHcl(deploymentFreeze octopus.DeploymentFreeze, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if c.Excluder.IsResourceExcludedWithRegex(deploymentFreeze.Name,
		c.ExcludeAllDeploymentFreezes,
		c.ExcludeDeploymentFreezes,
		c.ExcludeDeploymentFreezesRegex,
		c.ExcludeDeploymentFreezesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + deploymentFreeze.Id)
		return nil
	}

	freezeName := "deploymentfreeze_" + sanitizer.SanitizeName(deploymentFreeze.Name)

	//if c.GenerateImportScripts {
	//	c.toBashImport(policyName, deploymentFreeze.Name, dependencies)
	//	c.toPowershellImport(policyName, deploymentFreeze.Name, dependencies)
	//}

	thisResource := data.ResourceDetails{}
	thisResource.Name = deploymentFreeze.Name
	thisResource.FileName = "space_population/" + freezeName + ".tf"
	thisResource.Id = deploymentFreeze.Id
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployDeploymentFreezeDataType + "." + freezeName + ".deployment_freezes) != 0 " +
			"? data." + octopusdeployDeploymentFreezeDataType + "." + freezeName + ".deployment_freezes[0].id " +
			": " + octopusdeployDeploymentFreezeResourceType + "." + freezeName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployDeploymentFreezeResourceType + "." + freezeName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployDeploymentFreezeResourceType + "." + freezeName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformDeploymentFreeze{
			Type:              octopusdeployDeploymentFreezeResourceType,
			Name:              freezeName,
			Id:                strutil.InputPointerIfEnabled(c.IncludeIds, &deploymentFreeze.Id),
			ResourceName:      deploymentFreeze.Name,
			Start:             deploymentFreeze.Start,
			End:               deploymentFreeze.End,
			RecurringSchedule: c.getRecurringSchedule(deploymentFreeze),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, deploymentFreeze, freezeName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployDeploymentFreezeDataType + "." + freezeName + ".deployment_freezes) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil

	}

	dependencies.AddResource(thisResource)

	if !stateless {
		dependencies.AddResource(c.toHclTenantScope(deploymentFreeze, thisResource.Lookup, dependencies)...)
		dependencies.AddResource(c.toHclProjectScope(deploymentFreeze, thisResource.Lookup, dependencies)...)
	}

	return nil
}

func (c DeploymentFreezeConverter) getRecurringSchedule(deploymentFreeze octopus.DeploymentFreeze) *terraform.TerraformDeploymentFreezeRecurringSchedule {
	if deploymentFreeze.RecurringSchedule == nil {
		return nil
	}

	return &terraform.TerraformDeploymentFreezeRecurringSchedule{
		EndType:             deploymentFreeze.RecurringSchedule.EndType,
		Type:                deploymentFreeze.RecurringSchedule.Type,
		Unit:                deploymentFreeze.RecurringSchedule.Unit,
		DateOfMonth:         deploymentFreeze.RecurringSchedule.DateOfMonth,
		DayNumberOfMonth:    deploymentFreeze.RecurringSchedule.DayNumberOfMonth,
		DayOfWeek:           deploymentFreeze.RecurringSchedule.DayOfWeek,
		DaysOfWeek:          deploymentFreeze.RecurringSchedule.DaysOfWeek,
		EndAfterOccurrences: deploymentFreeze.RecurringSchedule.EndAfterOccurrences,
		EndOnDate:           deploymentFreeze.RecurringSchedule.EndOnDate,
		MonthlyScheduleType: deploymentFreeze.RecurringSchedule.MonthlyScheduleType,
	}
}

func (c DeploymentFreezeConverter) toHclTenantScope(deploymentFreeze octopus.DeploymentFreeze, parentReference string, dependencies *data.ResourceDetailsCollection) []data.ResourceDetails {
	resources := []data.ResourceDetails{}

	for _, tenant := range deploymentFreeze.TenantProjectEnvironmentScope {
		freezeName := "deploymentfreezetenant_" + sanitizer.SanitizeName(deploymentFreeze.Name)

		thisResource := data.ResourceDetails{}
		thisResource.Name = deploymentFreeze.Name
		thisResource.FileName = "space_population/" + freezeName + ".tf"
		thisResource.Id = deploymentFreeze.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${" + octopusdeployDeploymentFreezeTenantResourceType + "." + freezeName + ".id}"

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformDeploymentFreezeTenant{
				Type:               octopusdeployDeploymentFreezeTenantResourceType,
				Name:               freezeName,
				Count:              nil,
				Id:                 strutil.InputPointerIfEnabled(c.IncludeIds, &deploymentFreeze.Id),
				DeploymentFreezeId: parentReference,
				EnvironmentId:      dependencies.GetResource("Environments", tenant.EnvironmentId),
				ProjectId:          dependencies.GetResource("Projects", tenant.ProjectId),
				TenantId:           dependencies.GetResource("Tenants", tenant.TenantId),
			}

			file := hclwrite.NewEmptyFile()

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		resources = append(resources, thisResource)
	}

	return resources
}

func (c DeploymentFreezeConverter) toHclProjectScope(deploymentFreeze octopus.DeploymentFreeze, parentReference string, dependencies *data.ResourceDetailsCollection) []data.ResourceDetails {

	resources := []data.ResourceDetails{}

	for projectId, projectEnvironment := range deploymentFreeze.ProjectEnvironmentScope {

		freezeName := "deploymentfreezeproject_" + sanitizer.SanitizeName(deploymentFreeze.Name)

		thisResource := data.ResourceDetails{}
		thisResource.Name = deploymentFreeze.Name
		thisResource.FileName = "space_population/" + freezeName + ".tf"
		thisResource.Id = deploymentFreeze.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${" + octopusdeployDeploymentFreezeProjectResourceType + "." + freezeName + ".id}"

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformDeploymentFreezeProject{
				Type:               octopusdeployDeploymentFreezeProjectResourceType,
				Name:               freezeName,
				Count:              nil,
				Id:                 strutil.InputPointerIfEnabled(c.IncludeIds, &deploymentFreeze.Id),
				DeploymentFreezeId: parentReference,
				ProjectId:          dependencies.GetResource("Projects", projectId),
				EnvironmentIds:     dependencies.GetResources("Environments", projectEnvironment...),
			}

			file := hclwrite.NewEmptyFile()

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		resources = append(resources, thisResource)
	}

	return resources
}

// writeData appends the data block for stateless modules
func (c DeploymentFreezeConverter) writeData(file *hclwrite.File, resource octopus.DeploymentFreeze, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c DeploymentFreezeConverter) buildData(resourceName string, resource octopus.DeploymentFreeze) terraform.TerraformDeploymentFreezeData {
	return terraform.TerraformDeploymentFreezeData{
		Type:        octopusdeployDeploymentFreezeDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}
