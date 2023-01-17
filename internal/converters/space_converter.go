package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"strings"
)

// SpaceConverter creates the files required to create a new space. These files are used in a separate
// terraform project, as you first need to a create a space, and then configure a second provider
// to use that space.
type SpaceConverter struct {
	Client client.OctopusClient
}

func (c SpaceConverter) ToHcl() (map[string]string, error) {

	spaceResourceName, spaceTf, err := c.createSpaceTf()

	if err != nil {
		return nil, err
	}

	results := map[string]string{
		"space_creation/space.tf": spaceTf,
	}

	// Generate space population common files
	commonProjectFiles := SpacePopulateCommonGenerator{}.ToHcl()

	// merge the maps
	for k, v := range commonProjectFiles {
		results["space_creation/"+k] = v
		results["space_population/"+k] = v
	}

	// Convert the feeds
	feeds, feedMap, err := FeedConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range feeds {
		results[k] = v
	}

	// Convert the accounts
	accounts, accountsMap, err := AccountConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range accounts {
		results[k] = v
	}

	// Convert the environments
	environments, environmentsMap, err := EnvironmentConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range environments {
		results[k] = v
	}

	// Convert the library variables
	variables, variableMap, templateMap, err := LibraryVariableSetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range variables {
		results[k] = v
	}

	// Convert the lifecycles
	lifecycles, lifecycleMap, err := LifecycleConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		EnvironmentsMap:   environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range lifecycles {
		results[k] = v
	}

	// Convert the worker pools
	pools, poolMap, err := WorkerPoolConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range pools {
		results[k] = v
	}

	// Convert the tag sets
	tagSets, _, err := TagSetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range tagSets {
		results[k] = v
	}

	// Convert the git credentials
	gitCredentials, _, err := GitCredentialsConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range gitCredentials {
		results[k] = v
	}

	// Convert the projects groups
	projects, projectsMap, projectsTemplateMap, err := ProjectGroupConverter{
		Client:                c.Client,
		SpaceResourceName:     spaceResourceName,
		FeedMap:               feedMap,
		LifecycleMap:          lifecycleMap,
		WorkPoolMap:           poolMap,
		AccountsMap:           accountsMap,
		LibraryVariableSetMap: variableMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range projects {
		results[k] = v
	}

	// Convert the tenants
	tenants, tenantsMap, err := TenantConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		EnvironmentsMap:   environmentsMap,
		ProjectsMap:       projectsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range tenants {
		results[k] = v
	}

	// Convert the certificates
	certificates, _, err := CertificateConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		EnvironmentsMap:   environmentsMap,
		TenantsMap:        tenantsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range certificates {
		results[k] = v
	}

	// Convert the tenant variables
	tenantVariables, err := TenantVariableConverter{
		Client:                c.Client,
		SpaceResourceName:     spaceResourceName,
		EnvironmentsMap:       environmentsMap,
		ProjectsMap:           projectsMap,
		LibraryVariableSetMap: variableMap,
		TenantsMap:            tenantsMap,
		ProjectTemplatesMap:   projectsTemplateMap,
		CommonTemplatesMap:    templateMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range tenantVariables {
		results[k] = v
	}

	// Convert the machine policies
	machinePolicies, machinePoliciesMap, err := MachinePolicyConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range machinePolicies {
		results[k] = v
	}

	// Convert the k8s targets
	k8sTargets, _, err := KubernetesTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		AccountMap:        accountsMap,
		EnvironmentMap:    environmentsMap,
		WorkerPoolMap:     poolMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range k8sTargets {
		results[k] = v
	}

	// Convert the ssh targets
	sshTargets, _, err := SshTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		AccountMap:        accountsMap,
		EnvironmentMap:    environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range sshTargets {
		results[k] = v
	}

	// Convert the ssh targets
	listeningTargets, _, err := ListeningTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		AccountMap:        accountsMap,
		EnvironmentMap:    environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range listeningTargets {
		results[k] = v
	}

	// Convert the polling targets
	pollingTargets, _, err := PollingTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		AccountMap:        accountsMap,
		EnvironmentMap:    environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range pollingTargets {
		results[k] = v
	}

	// Convert the cloud region targets
	cloudRegionTargets, _, err := CloudRegionTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		EnvironmentMap:    environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range cloudRegionTargets {
		results[k] = v
	}

	// Convert the cloud region targets
	offlineDropTargets, _, err := OfflineDropTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		EnvironmentMap:    environmentsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range offlineDropTargets {
		results[k] = v
	}

	// Convert the azure cloud service targets
	azureCloudServiceTargets, _, err := AzureCloudServiceTargetConverter{
		Client:            c.Client,
		SpaceResourceName: spaceResourceName,
		MachinePolicyMap:  machinePoliciesMap,
		EnvironmentMap:    environmentsMap,
		AccountMap:        accountsMap,
	}.ToHcl()

	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range azureCloudServiceTargets {
		results[k] = v
	}

	// Unescape dollar signs because of https://github.com/hashicorp/hcl/issues/323
	for k, v := range results {
		results[k] = strings.ReplaceAll(v, "$${", "${")
	}

	return results, nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) createSpaceTf() (string, string, error) {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return "", "", err
	}

	spaceResourceName := "octopus_space_" + util.SanitizeNamePointer(space.Name)
	spaceName := "${var.octopus_space_name}"

	terraformResource := terraform.TerraformSpace{
		Description:        space.Description,
		IsDefault:          space.IsDefault,
		IsTaskQueueStopped: space.TaskQueueStopped,
		Name:               spaceResourceName,
		//SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
		//SpaceManagersTeams:       space.SpaceManagersTeams,
		// TODO: import teams rather than defaulting to admins
		SpaceManagersTeams: []string{"teams-administrators"},
		ResourceName:       &spaceName,
		Type:               "octopusdeploy_space",
	}

	spaceOutput := terraform.TerraformOutput{
		Name:  "octopus_space_id",
		Value: "${octopusdeploy_space." + spaceResourceName + ".id}",
	}

	spaceNameVar := terraform.TerraformVariable{
		Name:        "octopus_space_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the new space (the exported space was called " + *space.Name + ")",
		Default:     space.Name,
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))

	block := gohcl.EncodeAsBlock(spaceNameVar, "variable")
	util.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return spaceResourceName, string(file.Bytes()), nil
}
