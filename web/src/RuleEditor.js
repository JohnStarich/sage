import React from 'react';
import './RuleEditor.css';

import Button from 'react-bootstrap/Button';
import Form from 'react-bootstrap/Form';
import RadioGroup from './RadioGroup';
import { cleanCategory } from './CategoryPicker';

export default function({ transaction, onClose }) {
  const [pattern, setPattern] = React.useState(null)
  const [precision, setPrecision] = React.useState('Fuzzy')
  const [customPattern, setCustomPattern] = React.useState(false)
  const [expensesOrRevenues, setExpensesOrRevenues] = React.useState("Both")
  if (!transaction) {
    return null
  }

  const originalPattern = RegExp.escape(transaction.Payee.toLocaleLowerCase())
  if (pattern === null) {
    setPattern(originalPattern)
    return null
  }

  if (! customPattern) {
    let computedPattern = originalPattern
    if (precision === 'Exact') {
      computedPattern = `,"${computedPattern}",`
    }
    const amountExpr = `\\d+(\\.\\d+)?`
    switch (expensesOrRevenues) {
      case "Expenses":
        computedPattern = `${computedPattern}.*,-${amountExpr},`
        break
      case "Revenues":
        computedPattern = `${computedPattern}.*,${amountExpr},`
        break
      default: // no added amount expression
    }
    if (computedPattern !== pattern) {
      setPattern(computedPattern)
      return null
    }
  }
  const ruleString = makeRuleString(transaction)
  const validPattern = pattern && pattern !== "" && validRegExp(pattern)
  const validRule = validRegExpAndMatches(pattern, ruleString)

  const expensesOrRevenuesChoices = ['Both']
  if (transaction.Postings[0].Amount <= 0) {
    expensesOrRevenuesChoices.push('Expenses')
  }
  if (transaction.Postings[0].Amount >= 0) {
    expensesOrRevenuesChoices.push('Revenues')
  }
  return (
    <div className="rule-editor">
      <p>Always set the category for "{transaction.Payee}" to <strong>{cleanCategory(transaction.Postings[1].Account)}</strong>?</p>

      <div>
      {customPattern
        ? (
          <>
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
                    <pre>{"date, payee, currency, amount, balance\n"+ruleString}</pre>
                  </>
                : null}
            </Form.Group>

            <Button variant="link" onClick={() => setCustomPattern(false)}>Back to express rule</Button>
          </>
        ) : (
          <>
            <RadioGroup
              choices={['Fuzzy', 'Exact']}
              defaultChoice={precision}
              label="How precise?"
              onSelect={choice => setPrecision(choice)}
            />

            <RadioGroup
              choices={expensesOrRevenuesChoices}
              defaultChoice={expensesOrRevenues}
              label="Expenses or revenues?"
              onSelect={choice => setExpensesOrRevenues(choice)}
            />

            <Button variant="link" onClick={() => setCustomPattern(true)}>Advanced rule</Button>
          </>
        )
      }
      </div>

      <div className="rule-controls">
        <Button variant="primary" disabled onClick={() => alert("updated!")}>Save</Button>
        <Button variant="outline-danger" onClick={onClose}>Cancel</Button>
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

function makeRuleString(transaction) {
  return [
    transaction.Date,
    JSON.stringify(transaction.Payee),
    transaction.Postings[0].Currency,
    transaction.Postings[0].Amount,
    0, // balance
  ].join(',')
}
