package client

import (
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

func AccountsFromOFXClientIni(fileName string) ([]Account, error) {
	var accounts []Account
	cfg, err := ini.Load(fileName)
	if err != nil {
		return nil, err
	}

	for _, section := range cfg.Sections() {
		if section.Name() == ini.DEFAULT_SECTION {
			continue
		}
		getErr := false
		field := ""
		mustGet := func(key string) string {
			value := section.Key(key).String()
			if value == "" {
				getErr = true
				field = key
			}
			return value
		}

		inst := NewInstitution(
			mustGet("institution.description"),
			mustGet("institution.id"),
			mustGet("institution.org"),
			mustGet("institution.url"),
			mustGet("institution.username"),
			mustGet("institution.password"),
			Config{
				AppID:      mustGet("institution.client_args.app_id"),
				AppVersion: mustGet("institution.client_args.app_version"),
				OFXVersion: mustGet("institution.client_args.ofx_version"),
			},
		)
		if accountType := section.Key("account_type").String(); accountType != "" {
			// bank
			switch accountType {
			case "CHECKING":
				accounts = append(accounts, NewCheckingAccount(
					mustGet("number"),
					mustGet("routing_number"),
					inst,
				))
			case "SAVINGS":
				accounts = append(accounts, NewSavingsAccount(
					mustGet("number"),
					mustGet("routing_number"),
					inst,
				))
			default:
				return nil, errors.Errorf("Unknown account type: %s", accountType)
			}
		} else {
			// credit card
			accounts = append(accounts, NewCreditCard(mustGet("number"), inst))
		}
		if getErr {
			return nil, errors.New("Failed to parse ofxclient.ini: missing field: " + field)
		}
	}

	return accounts, nil
}
