package ledger

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Posting struct {
	Account  string
	Amount   *decimal.Decimal
	Balance  *decimal.Decimal
	Comment  string
	Currency string
	Tags     map[string]string
}

func NewPostingFromString(line string) (Posting, error) {
	var posting Posting
	// TODO support more than USD
	posting.Currency = "$"
	// comment / tags
	tokens := strings.SplitN(line, ";", 2)
	if len(tokens) == 0 {
		return posting, fmt.Errorf("Invalid posting has no tokens: '%s'", line)
	}
	line = strings.TrimSpace(tokens[0])
	if len(tokens) == 2 {
		posting.Comment, posting.Tags = parseTags(strings.TrimSpace(tokens[1]))
	}

	// account
	tokens = strings.SplitN(line, "  ", 2)
	if len(tokens) == 0 {
		return posting, fmt.Errorf("An account name must be specified: '%s'", line)
	}
	posting.Account = strings.TrimSpace(tokens[0])
	if len(tokens) < 2 {
		return posting, nil
	}
	line = strings.TrimSpace(tokens[1])

	// amount / balance
	tokens = strings.SplitN(line, "=", 2)
	if len(tokens) == 0 {
		return posting, fmt.Errorf("Invalid posting amount: '%s'", line)
	}
	var err error
	posting.Amount, err = parseAmount(strings.TrimSpace(tokens[0]))
	if err != nil {
		return posting, errors.Wrap(err, "Invalid amount")
	}
	if len(tokens) == 2 {
		posting.Balance, err = parseAmount(strings.TrimSpace(tokens[1]))
		if err != nil {
			return posting, errors.Wrap(err, "Invalid balance")
		}
	}
	return posting, nil
}

func parseAmount(amount string) (*decimal.Decimal, error) {
	amount = strings.TrimPrefix(amount, "$")
	amount = strings.TrimSpace(amount)
	decAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, err
	}
	return &decAmount, nil
}

func (p Posting) ID() string {
	return p.Tags[idTag]
}

func stringPad(s string, amount int) string {
	formatString := fmt.Sprintf("%%%ds", amount)
	return fmt.Sprintf(formatString, s)
}

func (p Posting) FormatTable(accountLen, amountLen int) string {
	var amount string
	if p.Amount != nil {
		amount = fmt.Sprintf("%s %s", p.Currency, p.Amount.String())
		amount = stringPad(amount, amountLen+len(p.Currency)+1)
	}
	var balance string
	if p.Balance != nil {
		balance = fmt.Sprintf(" = %s %s", p.Currency, p.Balance.String())
	}
	return fmt.Sprintf(
		"%s  %s%s%s\n",
		stringPad(p.Account, accountLen),
		amount,
		balance,
		serializeComment(p.Comment, p.Tags),
	)
}

func (p Posting) String() string {
	return p.FormatTable(1, 1)
}
