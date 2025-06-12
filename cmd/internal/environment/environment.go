package environment

import "os"

func GetPort() string {
	// Get the port from the environment variable
	port := os.Getenv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if port == "" {
		port = os.Getenv("OCTOTERRA_FUNCTIONS_CUSTOMHANDLER_PORT")
		if port == "" {
			port = "8080" // Default port
		}
	}
	return port
}
