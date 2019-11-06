package drivers

import (
	"time"

	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/client/web"
	"github.com/pkg/errors"
)

var driverFuncs = make(map[string]Driver)

// Connect creates a Requestor with the given driver name and connection information
func Connect(name string, connector Connector) (Requestor, error) {
	driver, exists := driverFuncs[name]
	if !exists {
		return nil, errors.Errorf("Driver does not exist with name: %s", name)
	}
	return driver(connector)
}

// Register adds a driver with the given name to the registry. Enables a call with Connect and the same driver name
func Register(name string, driver Driver) {
	if _, exists := driverFuncs[name]; exists {
		panic("Driver with duplicate name registered: " + name)
	}
	driverFuncs[name] = driver
}

// Driver creates a Requestor from the given connection details
type Driver func(Connector) (Requestor, error)

// Requestor downloads statements from an institution's website
type Requestor interface {
	// Statement downloads transactions with browser between the start and end times
	Statement(browser web.Browser, start, end time.Time) (*ofxgo.Response, error)
}
