package direct

import "strings"

type Driver interface {
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

var directConnectInstitutions []Driver

func Register(drivers ...Driver) {
	if len(directConnectInstitutions) == 0 {
		directConnectInstitutions = make([]Driver, 0, len(drivers))
	}
	for _, driver := range drivers {
		for _, support := range driver.MessageSupport() {
			switch support {
			case MessageBank, MessageCreditCard:
				directConnectInstitutions = append(directConnectInstitutions, driver)
				return
			}
		}
	}
}

func Search(query string) []Driver {
	var results []Driver
	for _, driver := range directConnectInstitutions {
		if strings.Contains(driver.Description(), query) {
			results = append(results, driver)
		}
	}
	return results
}
