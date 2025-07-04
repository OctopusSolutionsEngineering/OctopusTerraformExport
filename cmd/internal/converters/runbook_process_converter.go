package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"go.uber.org/zap"
	"net/url"
)

// RunbookProcessConverter converts deployment processes for v1 of the Octopus Terraform provider.
type RunbookProcessConverter struct {
	DeploymentProcessConverterBase
}

func (c *RunbookProcessConverter) ToHclByIdBranchAndProject(parentId string, runbookProcessId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdBranchAndProject(parentId, runbookProcessId, branch, recursive, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclStatelessByIdBranchAndProject(parentId string, runbookProcessId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdBranchAndProject(parentId, runbookProcessId, branch, true, true, dependencies)
}

func (c *RunbookProcessConverter) toHclByIdBranchAndProject(parentId string, runbookProcessId string, branch string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/runbookProcesses/"+runbookProcessId, &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	found, err = c.Client.GetSpaceResourceById("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	if !found {
		return fmt.Errorf("runbook with ID %s not found in project %s", resource.RunbookId, parentId+"/"+branch)
	}

	project := octopus.Project{}
	found, err = c.Client.GetSpaceResourceById("Projects", parentId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	if !found {
		return fmt.Errorf("project with ID %s not found", parentId)
	}

	return c.toHcl(&resource, &project, &runbook, recursive, false, stateless, dependencies)
}

func (c *RunbookProcessConverter) ToHclLookupByIdBranchAndProject(parentId string, runbookProcessId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/runbookProcesses/"+runbookProcessId, &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(&resource, &project, &runbook, false, true, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, false, dependencies)
}

func (c *RunbookProcessConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, true, dependencies)
}

func (c *RunbookProcessConverter) toHclById(id string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	zap.L().Info("Deployment Process: " + resource.Id)
	return c.toHcl(&resource, &project, &runbook, recursive, false, stateless, dependencies)
}

func (c *RunbookProcessConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", runbook.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(&resource, &project, &runbook, false, true, false, dependencies)
}
