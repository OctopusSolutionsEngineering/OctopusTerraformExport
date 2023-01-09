package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

type DeploymentProcessConverter struct {
	Client        client.OctopusClient
	FeedMap       map[string]string
	WorkPoolMap   map[string]string
	AccountsMap   map[string]string
	ProjectLookup string
}

func (c DeploymentProcessConverter) ToHclById(id string) (map[string]string, string, error) {
	resource := octopus.DeploymentProcess{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, "", err
	}

	resourceName := "deployment_process_" + util.SanitizeName(c.ProjectLookup)

	terraformResource := terraform.TerraformDeploymentProcess{
		Type:      "octopusdeploy_deployment_process",
		Name:      resourceName,
		ProjectId: c.ProjectLookup,
		Step:      make([]terraform.TerraformStep, len(resource.Steps)),
	}

	for i, s := range resource.Steps {
		terraformResource.Step[i] = terraform.TerraformStep{
			Name:               s.Name,
			PackageRequirement: s.PackageRequirement,
			Properties:         c.replaceFeedIds(s.Properties),
			Condition:          s.Condition,
			StartTrigger:       s.StartTrigger,
			Action:             make([]terraform.TerraformAction, len(s.Actions)),
		}

		for j, a := range s.Actions {

			terraformResource.Step[i].Action[j] = terraform.TerraformAction{
				Name:                          a.Name,
				ActionType:                    a.ActionType,
				Notes:                         a.Notes,
				IsDisabled:                    a.IsDisabled,
				CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
				IsRequired:                    a.IsRequired,
				WorkerPoolId:                  c.WorkPoolMap[a.WorkerPoolId],
				Container:                     c.convertContainer(a.Container),
				WorkerPoolVariable:            a.WorkerPoolVariable,
				Environments:                  a.Environments,
				ExcludedEnvironments:          a.ExcludedEnvironments,
				Channels:                      a.Channels,
				TenantTags:                    a.TenantTags,
				Package:                       make([]terraform.TerraformPackage, len(a.Packages)),
				Condition:                     a.Condition,
				RunOnServer:                   c.getRunOnServer(a.Properties),
				Properties:                    c.removeUnnecessaryFields(c.replaceIds(util.SanitizeMap(a.Properties))),
			}

			for k, p := range a.Packages {
				feedVar := c.FeedMap[*p.FeedId]
				terraformResource.Step[i].Action[j].Package[k] = terraform.TerraformPackage{
					Name:                    p.Name,
					PackageID:               p.PackageId,
					AcquisitionLocation:     p.AcquisitionLocation,
					ExtractDuringDeployment: p.ExtractDuringDeployment,
					FeedId:                  &feedVar,
					Id:                      p.Id,
					Properties:              c.replaceIds(p.Properties),
				}
			}
		}
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
		"space_population/" + resourceName + ".tf": string(file.Bytes()),
	}, resourceName, nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c DeploymentProcessConverter) convertContainer(container octopus.Container) *terraform.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformContainer{
			FeedId: container.FeedId,
			Image:  container.Image,
		}
	}

	return nil
}

func (c DeploymentProcessConverter) replaceIds(properties map[string]string) map[string]string {
	return c.replaceAccountIds(
		c.replaceAccountIds(properties))
}

// removeUnnecessaryFields removes generic property bag values that have more specific terraform properties
func (c DeploymentProcessConverter) removeUnnecessaryFields(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if k != "Octopus.Action.RunOnServer" {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

func (c DeploymentProcessConverter) getRunOnServer(properties map[string]any) bool {
	v, ok := properties["Octopus.Action.RunOnServer"]
	if ok {
		return strings.ToLower(fmt.Sprint(v)) == "true"
	}

	return true
}

// replaceFeedIds looks for any property value that is a valid feed ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c DeploymentProcessConverter) replaceFeedIds(properties map[string]string) map[string]string {
	for k, v := range properties {
		for k2, v2 := range c.FeedMap {
			if strings.Contains(v, k2) {
				properties[k] = strings.ReplaceAll(v, k2, v2)
			}
		}
	}

	return properties
}

// replaceAccountIds looks for any property value that is a valid account ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c DeploymentProcessConverter) replaceAccountIds(properties map[string]string) map[string]string {
	for k, v := range properties {
		for k2, v2 := range c.AccountsMap {
			if strings.Contains(v, k2) {
				properties[k] = strings.ReplaceAll(v, k2, v2)
			}
		}
	}

	return properties
}
