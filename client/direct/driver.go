package direct

import (
	"github.com/johnstarich/sage/search"
)

type Driver interface {
	ID() string
	Description() string
	FID() string
	Org() string
	URL() string
	MessageSupport() []DriverMessage
}

type DriverMessage int

const (
	MessageSignon DriverMessage = iota + 1
	MessageBank
	MessageCreditCard
)

var directConnectInstitutions = make(map[string]Driver)

func Register(drivers ...Driver) {
	if len(directConnectInstitutions) == 0 {
		directConnectInstitutions = make(map[string]Driver, len(drivers))
	}
	for _, driver := range drivers {
		if supportedDriver(driver) {
			directConnectInstitutions[driver.ID()] = driver
		}
	}
}

func supportedDriver(d Driver) bool {
	for _, support := range d.MessageSupport() {
		switch support {
		case MessageBank, MessageCreditCard:
			return true
		}
	}
	return false
}

func Search(query string) []Driver {
	driverNames := make([]string, 0, len(directConnectInstitutions))
	drivers := make([]Driver, 0, len(directConnectInstitutions))
	for _, driver := range directConnectInstitutions {
		drivers = append(drivers, driver)
		driverNames = append(driverNames, driver.Description())
	}
	resultIndexes := search.QueryIndexes(driverNames, query)

	results := make([]Driver, 0, len(resultIndexes))
	for _, ix := range resultIndexes {
		results = append(results, drivers[ix])
	}
	return results
}
