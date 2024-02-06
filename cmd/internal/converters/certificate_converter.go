package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployCertificateDataType = "octopusdeploy_certificates"
const octopusdeployCertificateResourceType = "octopusdeploy_certificate"

type CertificateConverter struct {
	Client                    client.OctopusClient
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           TagSetConverter
}

func (c CertificateConverter) AllToHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c CertificateConverter) AllToStatelessHcl(dependencies *ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c CertificateConverter) allToHcl(stateless bool, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Certificate]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Certificate: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c CertificateConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Certificate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Certificate: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c CertificateConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	certificate := octopus.Certificate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &certificate)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(certificateName, certificate)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a certificate called \""+certificate.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.certificates) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) buildData(resourceName string, resource octopus.Certificate) terraform.TerraformCertificateData {
	return terraform.TerraformCertificateData{
		Type:        octopusdeployCertificateDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c CertificateConverter) writeData(file *hclwrite.File, resource octopus.Certificate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c CertificateConverter) toHcl(certificate octopus.Certificate, recursive bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	/*
		Note we don't export the tenants or environments that this certificate might be exposed to.
		It is assumed the exported project links up all required environments, and the certificate
		will link itself to any available environments or tenants.
	*/

	certificateName := "certificate_" + sanitizer.SanitizeName(certificate.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + certificateName + ".tf"
	thisResource.Id = certificate.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployCertificateResourceType + "." + certificateName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates) != 0 " +
			"? data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates[0].id " +
			": " + octopusdeployCertificateResourceType + "." + certificateName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployCertificateResourceType + "." + certificateName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployCertificateResourceType + "." + certificateName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		err := c.writeMainResource(file, certificateName, certificate, recursive, stateless, dependencies)

		if err != nil {
			return "", err
		}

		err = c.writeVariables(file, certificateName, certificate)

		if err != nil {
			return "", err
		}

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c CertificateConverter) writeVariables(file *hclwrite.File, certificateName string, certificate octopus.Certificate) error {

	defaultPassword := ""
	certificatePassword := terraform.TerraformVariable{
		Name:        certificateName + "_password",
		Type:        "string",
		Nullable:    true,
		Sensitive:   true,
		Description: "The password used by the certificate " + certificate.Name,
		Default:     &defaultPassword,
	}

	if c.DummySecretVariableValues {
		certificatePassword.Default = c.DummySecretGenerator.GetDummyCertificatePassword()
	}

	block := gohcl.EncodeAsBlock(certificatePassword, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	certificateData := terraform.TerraformVariable{
		Name:        certificateName + "_data",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The certificate data used by the certificate " + certificate.Name,
	}

	if c.DummySecretVariableValues {
		certificateData.Default = c.DummySecretGenerator.GetDummyCertificate()
	}

	block = gohcl.EncodeAsBlock(certificateData, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return nil
}

func (c CertificateConverter) writeMainResource(file *hclwrite.File, certificateName string, certificate octopus.Certificate, recursive bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	terraformResource := terraform.TerraformCertificate{
		Type:            octopusdeployCertificateResourceType,
		Name:            certificateName,
		SpaceId:         nil,
		ResourceName:    certificate.Name,
		Password:        "${var." + certificateName + "_password}",
		CertificateData: "${var." + certificateName + "_data}",
		Archived:        &certificate.Archived,
		//CertificateDataFormat:           certificate.CertificateDataFormat,
		Environments: c.lookupEnvironments(certificate.EnvironmentIds, dependencies),
		//HasPrivateKey:                   certificate.HasPrivateKey,
		//IsExpired:                       certificate.IsExpired,
		//IssuerCommonName:                certificate.IssuerCommonName,
		//IssuerDistinguishedName:         certificate.IssuerDistinguishedName,
		//IssuerOrganization:              certificate.IssuerOrganization,
		//NotAfter:                        certificate.NotAfter,
		//NotBefore:                       certificate.NotBefore,
		Notes: &certificate.Notes,
		//ReplacedBy:                      nil, // ReplacedBy does not seem to be used
		//SelfSigned:                      certificate.SelfSigned,
		//SerialNumber:                    certificate.SerialNumber,
		//SignatureAlgorithmName:          certificate.SignatureAlgorithmName,
		//SubjectAlternativeNames:         certificate.SubjectAlternativeNames,
		//SubjectCommonName:               certificate.SubjectCommonName,
		//SubjectDistinguishedName:        certificate.SubjectDistinguishedName,
		//SubjectOrganization:             certificate.SubjectOrganization,
		TenantTags:                      c.Excluder.FilteredTenantTags(certificate.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
		TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
		Tenants:                         c.lookupTenants(certificate.TenantIds, dependencies),
		//Thumbprint:                      certificate.Thumbprint,
		//Version:                         certificate.Version,
	}

	if stateless {
		c.writeData(file, certificate, certificateName)
		terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployCertificateDataType + "." + certificateName + ".certificates) != 0 ? 0 : 1}")
	}

	// Add a comment with the import command
	baseUrl, _ := c.Client.GetSpaceBaseUrl()
	file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), certificate.Name, octopusdeployCertificateResourceType, certificateName))

	targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
	err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)

	// When using dummy values, we expect the secrets will be updated later
	if c.DummySecretVariableValues || stateless {

		ignoreAll := terraform.EmptyBlock{}
		lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
		targetBlock.Body().AppendBlock(lifecycleBlock)

		if c.DummySecretVariableValues {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[password, certificate_data]")
		}

		if stateless {
			hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
		}
	}

	if err != nil {
		return err
	}

	file.Body().AppendBlock(targetBlock)

	return nil
}

func (c CertificateConverter) GetResourceType() string {
	return "Certificates"
}

func (c CertificateConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, 0)
	for _, v := range envs {
		environment := dependencies.GetResource("Environments", v)
		if environment != "" {
			newEnvs = append(newEnvs, environment)
		}
	}
	return newEnvs
}

func (c CertificateConverter) lookupTenants(tenants []string, dependencies *ResourceDetailsCollection) []string {
	newTenants := make([]string, 0)
	for _, v := range tenants {
		tenant := dependencies.GetResource("Tenants", v)
		if tenant != "" {
			newTenants = append(newTenants, tenant)
		}
	}
	return newTenants
}
