package octopus

// Machine is a minimal representation capturing the common fields required to distinguish and identify a target
type Machine struct {
	NameId
	Endpoint MachineEndpointResource
}

type MachineEndpointResource struct {
	CommunicationStyle string
}
