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
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/url"
	"strings"
)

const octopusdeployChannelDataType = "octopusdeploy_channels"
const octopusdeployChannelResourceType = "octopusdeploy_channel"

type ChannelConverter struct {
	Client                   client.OctopusClient
	LifecycleConverter       ConverterAndLookupWithStatelessById
	ExcludeTenantTags        args.StringSliceArgs
	ExcludeTenantTagSets     args.StringSliceArgs
	Excluder                 ExcludeByName
	ErrGroup                 *errgroup.Group
	IncludeIds               bool
	LimitResourceCount       int
	IncludeDefaultChannel    bool
	IncludeSpaceInPopulation bool
	IgnoreCacErrors          bool
	ExcludeAllChannels       bool
	ExcludeChannels          args.StringSliceArgs
	ExcludeChannelsRegex     args.StringSliceArgs
	ExcludeChannelsExcept    args.StringSliceArgs
	ExcludeInvalidChannels   bool
}

func (c ChannelConverter) ToHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdWithTerraDependencies(projectId, terraformDependencies, false, dependencies)
}

func (c ChannelConverter) ToHclStatelessByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByProjectIdWithTerraDependencies(projectId, terraformDependencies, true, dependencies)
}

func (c ChannelConverter) toHclByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Channel]: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	for _, resource := range collection.Items {
		if dependencies.HasResource(resource.Id, c.GetResourceType()) {
			continue
		}

		zap.L().Info("Channel: " + resource.Id + " " + resource.Name)

		err = c.toHcl(resource, project, true, false, stateless, terraformDependencies, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// ToHclLookupByProjectIdWithTerraDependencies exports the channel set as a complete resource, but will reference external resources like
// lifecycles as data source lookups.
func (c ChannelConverter) ToHclLookupByProjectIdWithTerraDependencies(projectId string, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Channel]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Channel]: %w", err)
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", projectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	for _, resource := range collection.Items {
		if dependencies.HasResource(resource.Id, c.GetResourceType()) {
			continue
		}

		zap.L().Info("Channel: " + resource.Id + " " + resource.Name)
		err = c.toHcl(resource, project, false, true, false, terraformDependencies, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// We consider channels to be the responsibility of a project. If the project exists, we don't create the channel.
func (c ChannelConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
	return terraform.TerraformProjectData{
		Type:        octopusdeployProjectsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ChannelConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ChannelConverter) toHcl(channel octopus.Channel, project octopus.Project, recursive bool, lookup bool, stateless bool, terraformDependencies map[string]string, dependencies *data.ResourceDetailsCollection) error {
	if c.Excluder.IsResourceExcludedWithRegex(channel.Name, c.ExcludeAllChannels, c.ExcludeChannels, c.ExcludeChannelsRegex, c.ExcludeChannelsExcept) {
		return nil
	}

	/*
		It is possible to define a channel that references deployment steps or packages that no longer exist. This option
		allows us to skip channels that are invalid.
	*/
	if c.ExcludeInvalidChannels {
		invalid, err := c.isInvalid(channel, project)

		if err != nil {
			return err
		}

		if invalid {
			zap.L().Info("Channel " + channel.Name + " (" + channel.Id + ") is invalid - skipping ")
			return nil
		}
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + channel.Id)
		return nil
	}

	if channel.LifecycleId != "" {
		var err error
		if recursive {
			if stateless {
				err = c.LifecycleConverter.ToHclStatelessById(channel.LifecycleId, dependencies)
			} else {
				err = c.LifecycleConverter.ToHclById(channel.LifecycleId, dependencies)
			}

		} else if lookup {
			err = c.LifecycleConverter.ToHclLookupById(channel.LifecycleId, dependencies)
		}

		if err != nil {
			return err
		}
	}

	// CaC projects use action slugs to reference deployment actions in package rules.
	// Terraform needs the action names. So we need to look up the deployment process to get the action slugs and convert
	// then to action names
	var resource *octopus.DeploymentProcess = nil
	if project.HasCacConfigured() {
		resource = &octopus.DeploymentProcess{}
		_, err := c.Client.GetResource("Projects/"+project.Id+"/"+url.QueryEscape(project.PersistenceSettings.DefaultBranch)+"/deploymentprocesses", resource)
		if err != nil && !c.IgnoreCacErrors {
			return err
		}
	}

	thisResource := data.ResourceDetails{}
	thisResource.Name = channel.Name
	resourceName := "channel_" + sanitizer.SanitizeName(project.Name) + "_" + sanitizer.SanitizeNamePointer(&channel.Name)
	projectResourceName := "project_" + sanitizer.SanitizeName(project.Name)
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = channel.Id
	thisResource.ResourceType = c.GetResourceType()

	if channel.Name == "Default" && !c.IncludeDefaultChannel {
		thisResource.Dependency = "${" + octopusdeployProjectResourceType + "." + projectResourceName + "}"

		// TODO: this is a hack.
		// The situation here is that we need to expose the default channel to new resources. However,
		// the project may not be created yet. What we really need is a way to depend on the default channel
		// of a project resource, but this is not exposed. Even if the channel data lookup was project
		// specific, data blocks are queried before resources are created, so it wouldn't return
		// anything useful anyway.
		// In the event that we have resources that depend on the default channel (like project triggers),
		// they get this hard coded value. It isn't correct, but we need to provide something.
		// That said, triggers will create, even if the project does not own Channel-1.
		thisResource.Lookup = "Channels-1"

		// TODO: This needs to be project specific to mean anything
		thisResource.ToHcl = func() (string, error) {
			data := terraform.TerraformChannelData{
				Name:        resourceName,
				Type:        octopusdeployChannelDataType,
				Ids:         []string{},
				PartialName: channel.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			// Channel lookup really needs to be project specific before we can validate like this
			//hcl.WriteLifecyclePostCondition(block, "Failed to resolve a channel called \""+channel.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.channels) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		thisResource.Dependency = "${" + octopusdeployChannelResourceType + "." + resourceName + "}"

		if stateless {
			// TODO: because we can not retrieve a project specific channel from a data block, there is no good way
			// to construct a lookup here if the project exists. That said, if the project exists, no other resource
			// that might look up a channel (like project variables) will be created either, so nothing will ever use
			// the lookup. So we just use an empty string for the lookup.
			thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
				"? null " +
				": " + octopusdeployChannelResourceType + "." + resourceName + "[0].id}"
		} else {
			thisResource.Lookup = "${" + octopusdeployChannelResourceType + "." + resourceName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformChannel{
				Type:         octopusdeployChannelResourceType,
				Name:         resourceName,
				Id:           strutil.InputPointerIfEnabled(c.IncludeIds, &channel.Id),
				SpaceId:      strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", channel.SpaceId)),
				ResourceName: channel.Name,
				Description:  strutil.TrimPointer(channel.Description),
				LifecycleId:  c.getLifecycleId(channel.LifecycleId, dependencies),
				ProjectId:    dependencies.GetResource("Projects", channel.ProjectId),
				IsDefault:    channel.IsDefault,
				Rule:         c.convertRules(channel.Rules, resource),
				TenantTags:   c.Excluder.FilteredTenantTags(channel.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				// when importing a stateless project, the channel is only created if the project does not exist
				c.writeData(file, project.Name, resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + resourceName + ".projects) != 0 ? 0 : 1}")
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			/* Channels reference steps and packages by text without terraform understanding
			there is any relationship. In order for the channel to be created after the deployment process,
			we must make this dependency explicit. Otherwise, the channel may be created without the deployment
			process, and Octopus will reject the channel rules.*/
			manualDependencies := []string{}
			for resourceType, parentId := range terraformDependencies {
				if resourceType != "" && parentId != "" {
					stepDependencies := dependencies.GetResourceDependencyFromParent(parentId, resourceType)
					processedDependencies := lo.Map(stepDependencies, func(item string, index int) string {
						return hcl.RemoveId(hcl.RemoveInterpolation(item))
					})
					manualDependencies = append(manualDependencies, processedDependencies...)
				}
			}
			hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(manualDependencies[:], ",")+"]")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	}
	dependencies.AddResource(thisResource)
	return nil
}

func (c ChannelConverter) isInvalid(channel octopus.Channel, project octopus.Project) (bool, error) {
	resource := octopus.DeploymentProcess{}
	if project.HasCacConfigured() {
		if _, err := c.Client.GetResource("Projects/"+project.Id+"/"+url.QueryEscape(project.PersistenceSettings.DefaultBranch)+"/deploymentprocesses", &resource); err != nil {
			return true, err
		}
	} else {
		if _, err := c.Client.GetSpaceResourceById("DeploymentProcesses", strutil.EmptyIfNil(project.DeploymentProcessId), &resource); err != nil {
			return true, err
		}
	}

	for _, rule := range channel.Rules {
		for _, actionPackage := range rule.ActionPackages {
			_, stepExists := lo.Find(resource.Steps, func(item octopus.Step) bool {
				action, actionExists := lo.Find(item.Actions, func(item octopus.Action) bool {
					return item.Name == actionPackage.DeploymentAction
				})

				if !actionExists {
					_, packageExists := lo.Find(action.Packages, func(item octopus.Package) bool {
						return item.Name == actionPackage.DeploymentAction
					})

					return packageExists
				}

				return false
			})

			if !stepExists {
				return true, nil
			}
		}
	}

	return false, nil
}

func (c ChannelConverter) getLifecycleId(lifecycleId string, dependencies *data.ResourceDetailsCollection) *string {
	if lifecycleId == "" {
		return nil
	}

	lifecycleLookup := dependencies.GetResource("Lifecycles", lifecycleId)
	return &lifecycleLookup
}

func (c ChannelConverter) GetResourceType() string {
	return "Channels"
}

func (c ChannelConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/channels"
}

func (c ChannelConverter) convertRules(rules []octopus.Rule, deploymentprocess *octopus.DeploymentProcess) []terraform.TerraformRule {
	terraformRules := make([]terraform.TerraformRule, 0)
	for _, v := range rules {
		terraformRules = append(terraformRules, terraform.TerraformRule{
			ActionPackage: c.convertActionPackages(v.ActionPackages, deploymentprocess),
			Tag:           v.Tag,
			VersionRange:  v.VersionRange,
		})
	}
	return terraformRules
}

func (c ChannelConverter) convertActionPackages(actionPackages []octopus.ActionPackage, deploymentprocess *octopus.DeploymentProcess) []terraform.TerraformActionPackage {
	collection := make([]terraform.TerraformActionPackage, 0)
	for _, v := range actionPackages {

		// Assume the supplied deployment action is valid
		deploymentAction := v.DeploymentAction

		// However, if we have deployment process, we will check to see if the deployment action matches
		// an action slug. This is the case when converting a CaC project.
		if deploymentprocess != nil {
			actions := lo.FlatMap(deploymentprocess.Steps, func(item octopus.Step, index int) []octopus.Action {
				return item.Actions
			})
			action := lo.Filter(actions, func(item octopus.Action, index int) bool {
				return item.Slug != nil && *item.Slug == *deploymentAction
			})

			if len(action) == 1 {
				deploymentAction = action[0].Name
			}
		}

		collection = append(collection, terraform.TerraformActionPackage{
			DeploymentAction: deploymentAction,
			PackageReference: v.PackageReference,
		})
	}
	return collection
}
