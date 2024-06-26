package octopus

type Certificate struct {
	Id                              string
	SpaceId                         string
	Name                            string
	Description                     *string
	TenantedDeploymentParticipation string
	EnvironmentIds                  []string
	TenantIds                       []string
	TenantTags                      []string
	CertificateDataFormat           string
	Archived                        string
	ReplacedBy                      string
	SubjectDistinguishedName        string
	SubjectCommonName               string
	SubjectOrganization             string
	IssuerDistinguishedName         string
	IssuerCommonName                string
	IssuerOrganization              string
	SelfSigned                      bool
	Thumbprint                      string
	NotAfter                        string
	NotBefore                       string
	Notes                           string
	IsExpired                       bool
	HasPrivateKey                   bool
	Version                         int
	SerialNumber                    string
	SignatureAlgorithmName          string
	SubjectAlternativeNames         []string
	CertificateChain                []CertificateChain
}

type CertificateChain struct {
	SubjectDistinguishedName string
	IssuerDistinguishedName  string
	Thumbprint               string
	NotAfter                 string
	NotBefore                string
	Version                  int
	SerialNumber             string
	SignatureAlgorithmName   string
}
