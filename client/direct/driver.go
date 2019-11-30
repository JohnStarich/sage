package direct

import (
	"strings"
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
	results := make([]Driver, 0)
	for _, driver := range directConnectInstitutions {
		if strings.Contains(driver.Description(), query) {
			results = append(results, driver)
		}
	}
	return results
}
