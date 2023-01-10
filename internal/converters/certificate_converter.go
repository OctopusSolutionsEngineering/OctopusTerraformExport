package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type CertificateConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	EnvironmentsMap   map[string]string
	TenantsMap        map[string]string
}

func (c CertificateConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Certificate]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, certificate := range collection.Items {
		certificateName := "certificate_" + util.SanitizeName(certificate.Name)

		terraformResource := terraform.TerraformCertificate{
			Type:            "octopusdeploy_certificate",
			Name:            certificateName,
			SpaceId:         nil,
			ResourceName:    certificate.Name,
			Password:        "${var." + certificateName + "_password}",
			CertificateData: "${var." + certificateName + "_data}",
			Archived:        &certificate.Archived,
			//CertificateDataFormat:           certificate.CertificateDataFormat,
			Environments: c.lookupEnvironments(certificate.EnvironmentIds),
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
			TenantTags:                      &certificate.TenantTags,
			TenantedDeploymentParticipation: &certificate.TenantedDeploymentParticipation,
			Tenants:                         c.lookupTenants(certificate.TenantIds),
			//Thumbprint:                      certificate.Thumbprint,
			//Version:                         certificate.Version,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		defaultPassword := ""
		certificatePassword := terraform.TerraformVariable{
			Name:        certificateName + "_password",
			Type:        "string",
			Nullable:    true,
			Sensitive:   true,
			Description: "The password used by the certificate " + certificate.Name,
			Default:     &defaultPassword,
		}

		block := gohcl.EncodeAsBlock(certificatePassword, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		certificateData := terraform.TerraformVariable{
			Name:        certificateName + "_data",
			Type:        "string",
			Nullable:    false,
			Sensitive:   true,
			Description: "The certificate data used by the certificate " + certificate.Name,
		}

		block = gohcl.EncodeAsBlock(certificateData, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		results["space_population/certificate_"+certificateName+".tf"] = string(file.Bytes())
		resultsMap[certificate.Id] = "${octopusdeploy_certificate." + certificateName + ".id}"
	}

	return results, resultsMap, nil
}

func (c CertificateConverter) GetResourceType() string {
	return "Certificates"
}

func (c CertificateConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentsMap[v]
	}
	return newEnvs
}

func (c CertificateConverter) lookupTenants(envs []string) []string {
	newTenants := make([]string, len(envs))
	for i, v := range envs {
		newTenants[i] = c.TenantsMap[v]
	}
	return newTenants
}
