package client

import (
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

type credConfig []map[string]string

func AccountsFromOFXClientINI(fileName string) ([]Account, error) {
	var cfg credConfig
	iniFile, err := ini.Load(fileName)
	if err != nil {
		return nil, err
	}

	for _, section := range iniFile.Sections() {
		if section.Name() != ini.DEFAULT_SECTION {
			cfg = append(cfg, section.KeysHash())
		}
	}

	return accountsFromOFXClientConfig(cfg)
}

func accountsFromOFXClientConfig(cfg credConfig) ([]Account, error) {
	var accounts []Account
	var errs credErrors

	for ix, section := range cfg {
		instDescription := section["institution.description"] + ":" + section["description"]
		mustGet := func(key string) string {
			value := section[key]
			if value == "" {
				errs = append(
					errs,
					errors.Errorf("Missing required field '%s' for ofxclient account #%d '%s'", key, ix+1, instDescription),
				)
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
		if accountType, ok := section["account_type"]; ok {
			// bank
			switch accountType {
			case checkingType:
				accounts = append(accounts, NewCheckingAccount(
					mustGet("number"),
					mustGet("routing_number"),
					section["description"],
					inst,
				))
			case savingsType:
				accounts = append(accounts, NewSavingsAccount(
					mustGet("number"),
					mustGet("routing_number"),
					section["description"],
					inst,
				))
			default:
				errs = append(errs, errors.Errorf("Unknown account type '%s' for ofxclient account #%d '%s'", accountType, ix+1, instDescription))
			}
		} else {
			// credit card
			accounts = append(accounts, NewCreditCard(mustGet("number"), section["description"], inst))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Wrap(errs, "Failed to parse ofxclient.ini")
	}
	return accounts, nil
}
