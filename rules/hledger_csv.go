package rules

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/johnstarich/sage/ledger"
	"github.com/pkg/errors"
)

type csvRule struct {
	Conditions []string // used for formatting purposes
	matchLine  *regexp.Regexp

	account1, Account2 string
	comment            string
}

func NewCSVRule(account1, account2, comment string, conditions ...string) (Rule, error) {
	conditions, pattern, err := validateConditions(conditions)
	if err != nil {
		return csvRule{}, err
	}
	return csvRule{
		Conditions: conditions,
		matchLine:  pattern,
		account1:   strings.TrimSpace(account1),
		Account2:   strings.TrimSpace(account2),
		comment:    strings.TrimSpace(comment),
	}, nil
}

func validateConditions(conditions []string) (cleanedConditions []string, re *regexp.Regexp, err error) {
	cleanedConditions = make([]string, 0, len(conditions))
	for _, c := range conditions {
		c = strings.TrimSpace(c)
		if c != "" {
			cleanedConditions = append(cleanedConditions, c)
		}
	}
	if len(cleanedConditions) == 0 {
		pattern, err := regexp.Compile("")
		return nil, pattern, err
	}
	pattern, err := regexp.Compile("(?i)" + strings.Join(cleanedConditions, "|"))
	return cleanedConditions, pattern, err
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
	if c.Account2 != "" {
		txn.Postings[1].Account = c.Account2
	}
	if c.comment != "" {
		comment := strings.Replace(c.comment, "%comment", txn.Postings[0].Comment, -1)
		txn.Postings[0].Comment = comment
	}
}

type csvRuleJSON csvRule

func (c *csvRule) UnmarshalJSON(data []byte) error {
	var jsonRule csvRuleJSON
	if err := json.Unmarshal(data, &jsonRule); err != nil {
		return err
	}
	*c = csvRule(jsonRule)
	conditions, pattern, err := validateConditions(c.Conditions)
	c.Conditions = conditions
	c.matchLine = pattern
	return err
}

func (c csvRule) String() string {
	var buf strings.Builder
	hasConditions := len(c.Conditions) > 0
	if hasConditions {
		buf.WriteString("if\n")
	}
	for _, cond := range c.Conditions {
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
	indent("account2", c.Account2)
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
