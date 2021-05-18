import React from 'react';
import './RuleEditor.css';

import API from './API';
import Button from 'react-bootstrap/Button';
import Form from 'react-bootstrap/Form';
import RadioGroup from './RadioGroup';
import { cleanCategory } from './CategoryPicker';


const amountExpr = `\\d+(\\.\\d+)?`

function payeeRegex(payee) {
  return RegExp.escape(payee.toLocaleLowerCase())
}

function unmarshalExpressChoices(expression, payee) {
  const originalPattern = payeeRegex(payee)
  const exact = expression.startsWith(`,"${originalPattern}",`)
  let revenues = true, expenses = true
  if (expression.includes(`.*,-${amountExpr},`)) {
    revenues = false
  } else if (expression.includes(`.*,${amountExpr},`)) {
    expenses = false
  }
  return { exact, expenses, revenues }
}

function marshalExpressChoices(choices, payee) {
  let expression = payeeRegex(payee)
  if (choices.exact) {
    expression = `,"${expression}",`
  }
  if (choices.expenses === choices.revenues) {
    // expenses and revenues shouldn't both be true, or the expression is already pretty weird
    return expression
  }
  if (choices.expenses) {
    return `${expression}.*,-${amountExpr},`
  }
  return `${expression}.*,${amountExpr},`
}

function extractExpRevChoice(choices) {
  if (choices.expenses !== choices.revenues) {
    return choices.expenses ? choiceExpenses : choiceRevenues
  }
  return choiceBoth
}

const choiceBoth = 'Both'
const choiceExpenses = 'Expenses'
const choiceRevenues = 'Revenues'
const choiceExact = 'Exact'
const choiceFuzzy = 'Fuzzy'

export default function RuleEditor({ transaction, onClose, rule, setRule, removeRule }) {
  if (!onClose) {
    throw Error("onClose is required")
  }
  if (!setRule) {
    throw Error("setRule is required")
  }
  if (!removeRule) {
    throw Error("removeRule is required")
  }

  const ruleString = React.useMemo(() => [
        transaction.Date,
        JSON.stringify(transaction.Payee),
        transaction.Postings[0].Currency,
        transaction.Postings[0].Amount,
        0, // balance
      ].join(','),
    [transaction])

  const conditionIndex = React.useMemo(() =>
      rule.Conditions.findIndex(cond =>
        new RegExp(cond, 'i').test(ruleString)),
    [ruleString, rule.Conditions])
  const condition = conditionIndex !== -1 ? rule.Conditions[conditionIndex] : null

  const [pattern, setPattern] = React.useState(() => condition || payeeRegex(transaction.Payee))
  const choices = React.useMemo(() => unmarshalExpressChoices(pattern, transaction.Payee), [pattern, transaction.Payee])
  const [customPattern, setCustomPattern] = React.useState(() =>
    pattern !== marshalExpressChoices(choices, transaction.Payee))
  const [precision, setPrecision] = React.useState(choices.exact ? choiceExact : choiceFuzzy)
  const [expensesOrRevenues, setExpensesOrRevenues] = React.useState(extractExpRevChoice(choices))

  React.useEffect(() => {
    // carry over information from express to advanced, or vice versa
    if (customPattern) {
      setPrecision(() => choices.exact ? choiceExact : choiceFuzzy)
      setExpensesOrRevenues(() => extractExpRevChoice(choices))
    } else {
      setPattern(() =>
        marshalExpressChoices({
          exact: precision === choiceExact,
          revenues: expensesOrRevenues === choiceRevenues,
          expenses: expensesOrRevenues === choiceExpenses,
        }, transaction.Payee))
    }
  }, [choices, customPattern, expensesOrRevenues, pattern, precision, transaction.Payee])

  const validPattern = pattern && pattern !== "" && validRegExp(pattern)
  const validRule = validRegExpAndMatches(pattern, ruleString)

  const disabledExpRevChoices = []
  if (transaction.Postings[0].Amount < 0) {
    disabledExpRevChoices.push(choiceRevenues)
  }
  if (transaction.Postings[0].Amount > 0) {
    disabledExpRevChoices.push(choiceExpenses)
  }

  const updateRule = async () => {
    let newConditions = rule.Conditions.slice()
    if (condition) {
      newConditions[conditionIndex] = pattern
    } else {
      newConditions.push(pattern)
    }
    const newRule = Object.assign({}, rule, {
      Conditions: newConditions,
      Index: rule.Index,
    })
    if (condition) {
      await API.post('/v1/updateRule', newRule)
    } else {
      const res = await API.post('/v1/addRule', newRule)
      newRule.Index = Number(res.data.Index)
    }
    setRule(newRule)
    onClose()
  }

  const deleteRule = async () => {
    let newConditions = rule.Conditions.slice()
    if (condition && newConditions.length === 1) {
      await API.post('/v1/deleteRule', { Index: rule.Index })
      removeRule()
      onClose()
      return
    }
    if (condition) {
      newConditions.splice(conditionIndex, 1)
    }
    const newRule = Object.assign({}, rule, {
      Conditions: newConditions,
      Index: rule.Index,
    })
    await API.post('/v1/updateRule', newRule)
    removeRule()
    onClose()
  }

  return (
    <div className="rule-editor">
      <Button variant="outline-secondary" onClick={onClose} className="rule-close">X</Button>
      <p>For new transactions, always categorize "{transaction.Payee}" as <strong>{cleanCategory(rule.Account2 || transaction.Postings[1].Account)}</strong>.</p>

      <div>
        {customPattern
          ? (
            <Form.Group>
              <Form.Label>Pattern</Form.Label>
              <Form.Control
                type="text"
                defaultValue={pattern}
                onChange={e => setPattern(e.target.value)}
                isValid={validPattern && validRule}
                isInvalid={!validPattern || !validRule}
              />
              {!validPattern ?
                <>
                  <br />
                  <div><em>Pattern is not a valid expression.</em></div>
                </>
                : null}
              {validPattern && !validRule ?
                <>
                  <br />
                  <div><em>Expression does not match this transaction:</em></div>
                  <br />
                  <pre>{"date, payee, currency, amount, balance\n" + ruleString}</pre>
                </>
                : null}
            </Form.Group>
          ) : (
            <>
              <RadioGroup
                choices={[choiceFuzzy, choiceExact]}
                defaultChoice={precision}
                label="How precise?"
                onSelect={choice => setPrecision(choice)}
              />

              <RadioGroup
                choices={[choiceBoth, choiceExpenses, choiceRevenues]}
                defaultChoice={expensesOrRevenues}
                disabledChoices={disabledExpRevChoices}
                label="Expenses or revenues?"
                onSelect={choice => setExpensesOrRevenues(choice)}
              />
            </>
          )
        }
      </div>

      <br />

      <div className="rule-controls">
        <Button variant="primary" disabled={!validPattern || !validRule} onClick={() => updateRule()}>{condition ? "Update" : "Add"}</Button>
        <Button variant="link" onClick={() => setCustomPattern(!customPattern)}>
          {customPattern ? "Back to express rule" : "Advanced rule"}
        </Button>
        {condition ?
          <Button variant="danger" onClick={() => {
            if (window.confirm(`Remove automatic category for ${transaction.Payee}?`)) {
              deleteRule()
            }
          }}>Remove</Button>
        : null}
      </div>
    </div>
  )
}


function validRegExp(pattern) {
  try {
    new RegExp(pattern, "i")
    return true
  } catch {
    return false
  }
}

function validRegExpAndMatches(pattern, intendedMatch) {
  try {
    const expr = new RegExp(pattern, "i")
    return expr.test(intendedMatch)
  } catch {
    return false
  }
}
