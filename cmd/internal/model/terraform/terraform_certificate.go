package terraform

type TerraformCertificate struct {
	Type                            string    `hcl:"type,label"`
	Name                            string    `hcl:"name,label"`
	Id                              *string   `hcl:"id"`
	Count                           *string   `hcl:"count"`
	SpaceId                         *string   `hcl:"space_id"`
	ResourceName                    string    `hcl:"name"`
	Password                        string    `hcl:"password"`
	CertificateData                 string    `hcl:"certificate_data"`
	Archived                        *string   `hcl:"archived"`
	CertificateDataFormat           *string   `hcl:"certificate_data_format"`
	Environments                    []string  `hcl:"environments"`
	HasPrivateKey                   *bool     `hcl:"has_private_key"`
	IsExpired                       *bool     `hcl:"is_expired"`
	IssuerCommonName                *string   `hcl:"issuer_common_name"`
	IssuerDistinguishedName         *string   `hcl:"issuer_distinguished_name"`
	IssuerOrganization              *string   `hcl:"issuer_organization"`
	NotAfter                        *string   `hcl:"not_after"`
	NotBefore                       *string   `hcl:"not_before"`
	Notes                           *string   `hcl:"notes"`
	ReplacedBy                      *string   `hcl:"replaced_by"`
	SelfSigned                      *bool     `hcl:"self_signed"`
	SerialNumber                    *string   `hcl:"serial_number"`
	SignatureAlgorithmName          *string   `hcl:"signature_algorithm_name"`
	SubjectAlternativeNames         *[]string `hcl:"subject_alternative_names"`
	SubjectCommonName               *string   `hcl:"subject_common_name"`
	SubjectDistinguishedName        *string   `hcl:"subject_distinguished_name"`
	SubjectOrganization             *string   `hcl:"subject_organization"`
	TenantTags                      []string  `hcl:"tenant_tags"`
	TenantedDeploymentParticipation *string   `hcl:"tenanted_deployment_participation"`
	Tenants                         []string  `hcl:"tenants"`
	Thumbprint                      *string   `hcl:"thumbprint"`
	Version                         *int      `hcl:"version"`
}
