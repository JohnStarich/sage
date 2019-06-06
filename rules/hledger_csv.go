package rules

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
)

type csvRule struct {
	conditions []string // used for formatting purposes
	matchLine  *regexp.Regexp

	account1, account2 string
	comment            string
}

func NewCSVRule(account1, account2, comment string, conditions ...string) (Rule, error) {
	pattern, err := regexp.Compile(strings.Join(conditions, "|"))
	if err != nil {
		return csvRule{}, err
	}
	return csvRule{
		conditions: conditions,
		matchLine:  pattern,
		account1:   account1,
		account2:   account2,
		comment:    comment,
	}, nil
}

// TODO add memoization?
// NOTE: assumes the transaction has at least one posting (i.e. a valid txn)
func ledgerMatchLine(txn ledger.Transaction) string {
	balance := ""
	if txn.Postings[0].Balance != nil {
		balance = txn.Postings[0].Balance.String()
	}
	return strings.Join([]string{
		txn.Date.Format(ledger.DateFormat),
		strconv.Quote(txn.Payee),
		txn.Postings[0].Currency,
		txn.Postings[0].Amount.String(),
		balance,
	}, ",")
}

func (c csvRule) Match(txn ledger.Transaction) bool {
	return c.matchLine.MatchString(ledgerMatchLine(txn))
}

func (c csvRule) Apply(txn *ledger.Transaction) {
	if c.account1 != "" {
		txn.Postings[0].Account = c.account1
	}
	if c.account2 != "" {
		txn.Postings[1].Account = c.account2
	}
	if c.comment != "" {
		comment := strings.Replace(c.comment, "%comment", txn.Postings[0].Comment, -1)
		txn.Postings[0].Comment = comment
	}
}

func (c csvRule) String() string {
	var buf strings.Builder
	hasConditions := len(c.conditions) > 0
	if hasConditions {
		buf.WriteString("if\n")
	}
	for _, cond := range c.conditions {
		buf.WriteString(cond)
		buf.WriteRune('\n')
	}

	indent := func(field, value string) {
		if value == "" {
			return
		}
		if hasConditions {
			buf.WriteString("  ")
		}
		buf.WriteString(field)
		buf.WriteRune(' ')
		buf.WriteString(value)
		buf.WriteRune('\n')
	}
	indent("account1", c.account1)
	indent("account2", c.account2)
	indent("comment", c.comment)

	return buf.String()
}

func NewCSVRulesFromReader(reader io.Reader) (Rules, error) {
	var rules Rules
	scanner := bufio.NewScanner(reader)

	var foundIf, foundExpressions bool
	var account1, account2, comment string
	var conditions []string
	endRule := func() error {
		if !foundExpressions {
			if foundIf {
				return errors.New("If statements must have a condition and expression")
			}
			// nothing found
			return nil
		}
		rule, err := NewCSVRule(account1, account2, comment, conditions...)
		if err != nil {
			return err
		}
		rules = append(rules, rule)
		foundIf = false
		foundExpressions = false
		account1 = ""
		account2 = ""
		comment = ""
		conditions = nil
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			// remove blank lines
			continue
		}

		if line == "if" || strings.HasPrefix(line, "if ") {
			if foundExpressions {
				if err := endRule(); err != nil {
					return nil, err
				}
			}
			if !foundIf {
				line = strings.TrimPrefix(line, "if")
			}
			foundIf = true
			line = strings.TrimSpace(line)
			if line != "" {
				conditions = append(conditions, line)
			}
		} else if foundIf && !strings.HasPrefix(line, " ") {
			conditions = append(conditions, line)
		} else {
			if foundIf && len(conditions) == 0 {
				return nil, errors.New("Started expressions but no conditions were found")
			}
			foundExpressions = true
			line = strings.TrimSpace(line)
			tokens := strings.SplitN(line, " ", 2)
			if len(tokens) != 2 {
				return nil, errors.Errorf("Rule transform line must have both key and value: '%s'", line)
			}
			key, value := tokens[0], tokens[1]
			value = strings.TrimSpace(value)
			switch key {
			case "account1":
				account1 = value
			case "account2":
				account2 = value
			case "comment":
				comment = value
			default:
				return nil, errors.Errorf("Unrecognized rule key: '%s'", key)
			}
		}
	}
	if err := endRule(); err != nil {
		return nil, err
	}

	return rules, nil
}
