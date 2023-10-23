package converters

// DummySecret is a simple service that returns the dummy value to use in place of a secret.
// This is required because the Octopus API never exports secrets, so octoterra can not
// include secrets in the exported Terraform module. It can be useful, however, to create resources that
// depend on secret values and update those secrets later. To do this, octoterra can optionally
// default secret values to dummy values.
type DummySecret struct {
}

func (e DummySecret) GetDummySecret() *string {
	retValue := "Change Me!"
	return &retValue
}
