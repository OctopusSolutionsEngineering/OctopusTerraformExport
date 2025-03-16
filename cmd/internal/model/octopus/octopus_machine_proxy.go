package octopus

type MachineProxy struct {
	Id        string
	Name      string
	Host      string
	Port      int
	ProxyType string
	Username  string
	Password  Secret
	SpaceId   string
}
