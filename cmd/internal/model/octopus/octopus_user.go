package octopus

type User struct {
	Id                  string
	username            string
	DisplayName         string
	IsActive            bool
	IsService           bool
	EmailAddress        string
	CanPasswordBeEdited bool
	IsRequestor         string
	Created             string
	Identities          []Identity
}

type Identity struct {
	IdentityProviderName string
	Claims               []Claim
}

type Claim struct {
	Email Email
	Dn    Dn
}

type Email struct {
	Value              string
	IsIdentifyingClaim bool
}

type Dn struct {
	Value              string
	IsIdentifyingClaim bool
}
