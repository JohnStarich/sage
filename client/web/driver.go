package web

import (
	"strings"

	"github.com/pkg/errors"
)

type Driver func(CredConnector) (Connector, error)

var driverFuncs = make(map[string]Driver)

// Connect creates a Connector with the given driver name and credentials
func Connect(connector CredConnector) (Connector, error) {
	name := strings.ToLower(connector.Driver())
	driver, exists := driverFuncs[name]
	if !exists {
		return nil, errors.Errorf("Driver does not exist with name: %s", name)
	}
	return driver(connector)
}

// Register adds a driver with the given name to the registry. Enables a call with Connect and the same driver name
func Register(name string, driver Driver) {
	name = strings.ToLower(name)
	if _, exists := driverFuncs[name]; exists {
		panic("Driver with duplicate name registered: " + name)
	}
	driverFuncs[name] = driver
}
