package ledger

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	usd = "$"
)

type Posting struct {
	Account  string
	Amount   decimal.Decimal
	Balance  *decimal.Decimal
	Comment  string
	Currency string
	Tags     map[string]string
}

func NewPostingFromString(line string) (Posting, error) {
	var posting Posting
	// TODO support more than USD
	// comment / tags
	tokens := strings.SplitN(line, ";", 2)
	line = strings.TrimSpace(tokens[0])
	if len(tokens) == 2 {
		posting.Comment, posting.Tags = parseTags(strings.TrimSpace(tokens[1]))
	}

	// account
	tokens = strings.SplitN(line, "  ", 2)
	posting.Account = strings.TrimSpace(tokens[0])
	if posting.Account == "" {
		return posting, fmt.Errorf("An account name must be specified: '%s'", line)
	}
	if len(tokens) < 2 {
		return posting, missingAmountErr
	}
	line = strings.TrimSpace(tokens[1])

	// amount / balance
	tokens = strings.SplitN(line, "=", 2)
	var err error
	posting.Amount, err = parseAmount(strings.TrimSpace(tokens[0]))
	if err != nil {
		return posting, errors.Wrap(err, "Invalid amount")
	}
	posting.Currency = usd
	if len(tokens) == 2 {
		var balance decimal.Decimal
		balance, err = parseAmount(strings.TrimSpace(tokens[1]))
		posting.Balance = &balance
		if err != nil {
			return posting, errors.Wrap(err, "Invalid balance")
		}
	}
	return posting, nil
}

func parseAmount(amount string) (decimal.Decimal, error) {
	amount = strings.TrimPrefix(amount, usd)
	amount = strings.TrimSpace(amount)
	// TODO support thousands delimiter other than ','
	amount = strings.Replace(amount, ",", "", -1)
	return decimal.NewFromString(amount)
}

func (p Posting) ID() string {
	return p.Tags[idTag]
}

func stringPad(s string, amount int) string {
	formatString := fmt.Sprintf("%%%ds", amount)
	return fmt.Sprintf(formatString, s)
}

func (p Posting) FormatTable(accountLen, amountLen int) string {
	amount := fmt.Sprintf("%s %s", p.Currency, p.Amount.String())
	amount = stringPad(amount, amountLen+len(p.Currency)+1)
	var balance string
	if p.Balance != nil {
		balance = fmt.Sprintf(" = %s %s", p.Currency, p.Balance.String())
	}
	return fmt.Sprintf(
		"%s  %s%s%s",
		stringPad(p.Account, accountLen),
		amount,
		balance,
		serializeComment(p.Comment, p.Tags),
	)
}

func (p Posting) String() string {
	return p.FormatTable(1, 1)
}
