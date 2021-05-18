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
	rule := csvRule{
		Conditions: conditions,
		matchLine:  pattern,
		account1:   strings.TrimSpace(account1),
		Account2:   strings.TrimSpace(account2),
		comment:    strings.TrimSpace(comment),
	}
	if rule.account1 == "" && rule.Account2 == "" && rule.comment == "" {
		return nil, errors.New("Invalid rule: No category selected")
	}
	return rule, nil
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
		pattern := regexp.MustCompile("")
		return nil, pattern, nil
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
		comment := strings.ReplaceAll(c.comment, "%comment", txn.Postings[0].Comment)
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

type readerState struct {
	foundIf            bool
	foundExpressions   bool
	account1, account2 string
	comment            string
	conditions         []string
}

func NewCSVRulesFromReader(reader io.Reader) (Rules, error) {
	var rules Rules
	scanner := bufio.NewScanner(reader)

	var state readerState

	endRule := func() error {
		if !state.foundExpressions {
			if state.foundIf {
				return errors.New("If statements must have a condition and expression")
			}
			// nothing found
			return nil
		}
		rule, err := NewCSVRule(state.account1, state.account2, state.comment, state.conditions...)
		if err != nil {
			return err
		}
		rules = append(rules, rule)
		state = readerState{}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			// remove blank lines
			continue
		}

		switch {
		case line == "if" || strings.HasPrefix(line, "if "):
			if err := foundIf(&state, line, endRule); err != nil {
				return nil, err
			}
		case state.foundIf && !strings.HasPrefix(line, " "):
			state.conditions = append(state.conditions, line)
		default:
			err := foundExpression(&state, line)
			if err != nil {
				return nil, err
			}
		}
	}
	if err := endRule(); err != nil {
		return nil, err
	}

	return rules, nil
}

func foundIf(state *readerState, line string, endRule func() error) error {
	if state.foundExpressions {
		if err := endRule(); err != nil {
			return err
		}
	}
	if !state.foundIf {
		line = strings.TrimPrefix(line, "if")
	}
	state.foundIf = true
	line = strings.TrimSpace(line)
	if line != "" {
		state.conditions = append(state.conditions, line)
	}
	return nil
}

func foundExpression(state *readerState, line string) error {
	if state.foundIf && len(state.conditions) == 0 {
		return errors.New("Started expressions but no conditions were found")
	}
	state.foundExpressions = true
	line = strings.TrimSpace(line)
	tokens := strings.SplitN(line, " ", 2)
	if len(tokens) != 2 {
		return errors.Errorf("Rule transform line must have both key and value: '%s'", line)
	}
	key, value := tokens[0], tokens[1]
	value = strings.TrimSpace(value)
	switch key {
	case "account1":
		state.account1 = value
	case "account2":
		state.account2 = value
	case "comment":
		state.comment = value
	default:
		return errors.Errorf("Unrecognized rule key: '%s'", key)
	}
	return nil
}
