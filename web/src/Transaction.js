import React from 'react';
import './Transaction.css';

import API from './API';
import Amount from './Amount';
import Button from 'react-bootstrap/Button';
import Form from 'react-bootstrap/Form';
import Modal from 'react-bootstrap/Modal';
import RuleEditor from './RuleEditor';
import Table from 'react-bootstrap/Table';
import { CategoryPicker, cleanCategory } from './CategoryPicker';


export default function Transaction(updateTransaction, accountIDMap) {
  return txn => {
    let postings = txn.Postings.map((p, i) => {
      if (i === 0) {
        let accountName
        if (accountIDMap[p.Account]) {
          accountName = accountIDMap[p.Account]
        } else {
          accountName = p.Account;
          let separatorIndex = accountName.indexOf(':')
          accountName = separatorIndex !== -1 ? accountName.slice(separatorIndex + 1) : accountName
          accountName = accountName.replace(/:/, " - ")
        }
        return Object.assign({}, p, { AccountName: accountName })
      }
      return p
    })

    const updatePosting = (index, newPosting) => {
      let { ID, Postings } = txn
      Postings = Array.from(Postings)
      Postings[index] = Object.assign({}, Postings[index], newPosting)
      Postings = balancePostings(Postings, index)
      updateTransaction({ ID, Postings })
    }

    return (
      <Form>
        <Table className="postings" responsive borderless>
          <tbody>
            {postings.map((posting, i) =>
              <tr key={posting.Account}>
                <td>
                  {i === 0
                    ? <Form.Control type="text" value={posting.AccountName} disabled />
                    : <CategoryPicker category={posting.Account} setCategory={c => updatePosting(i, { Account: c })} />
                  }
                </td>
                <td>
                  <Amount
                    amount={posting.Amount}
                    disabled={i === 0 || postings.length === 2}
                    editable
                    onChange={a => updatePosting(i, { Amount: a })}
                    prefix={posting.Currency}
                  />
                </td>
              </tr>
            )}
            <tr>
              <td>
                <TransactionRules
                  transaction={txn}
                  setCategory={category => updatePosting(postings.length-1, { Account: category })}
                  />
              </td>
            </tr>
          </tbody>
        </Table>
      </Form>
    )
  }
}

function balancePostings(postings, updatedIndex) {
  if (postings.length < 2) {
    // invalid transaction: must have at least 2 postings
    return postings
  }
  const delta =
    postings
      .map(p => p.Amount)
      .reduce((a, b) => a + b)
  if (delta === 0) {
    // transaction is balanced
    return postings
  }

  let leftOverIndex = updatedIndex + 1
  if (leftOverIndex === postings.length) {
    leftOverIndex = 1
  }
  postings[leftOverIndex].Amount = roundToHundredths(postings[leftOverIndex].Amount - delta)
  return postings
}

function roundToHundredths(x) {
  // espilon idea lifted from https://stackoverflow.com/a/11832950/1530494
  return Math.round((x + Number.EPSILON) * 100) / 100
}

function TransactionRules({ transaction, setCategory }) {
  if (!setCategory) {
    throw Error("setCategory is required")
  }

  const [editing, setEditing] = React.useState(null)
  const [rules, setRules] = React.useState([])
  const [rule, setRule] = React.useState(null)
  React.useEffect(() => {
    if (!transaction) {
      return
    }
    API.get('/v1/getRules', { params: { transaction: transaction.ID } })
      .then(res => {
        const rules = Object.entries(res.data.Rules)
          .map(([key, value]) =>
            Object.assign({ Index: Number(key) }, value)
          )
          .filter(r => r.Conditions && r.Conditions.length > 0)
          .sort((a, b) => a.Index - b.Index)
        if (rules && rules.length > 0) {
          setRules(rules)
          setRule(rules[rules.length - 1])
        } else {
          setRule(null)
        }
      })
  }, [transaction])

  const removeRule = () => {
    const newRules = rules.slice(0, -1)
    setRules(newRules)
    setRule(newRules.length > 0 ? newRules[newRules.length - 1] : null)
  }

  const account2 = transaction.Postings[transaction.Postings.length - 1].Account
  const isUncategorized = account2 === 'uncategorized' || account2 === 'expenses:uncategorized'

  return (
    <>
      {!isUncategorized && transaction.Postings.length === 2 ? (
        <>
          <Button
            className="edit-rule-btn"
            onClick={() => setEditing(rule || {Conditions: [], Account2: account2})}
            variant={rule ? "secondary" : "link"}
            >
            {rule
              ? "Edit rule"
              : <>Always categorize "{transaction.Payee}" as <strong>{cleanCategory(account2)}</strong>?</>
            }
          </Button>
          {rule && rule.Account2 && rule.Account2 !== account2 ?
            <Button variant="link" onClick={() => setCategory(rule.Account2)}>
              Use default category <strong>{cleanCategory(rule.Account2)}</strong>?
            </Button>
          : null}
        </>
      ) : null}
      <Modal show={editing !== null} onHide={() => setEditing(null)}>
        {editing !== null ? (
        <RuleEditor
          onClose={() => setEditing(null)}
          transaction={transaction}
          rule={editing}
          setRule={rule => {
            setRule(rule)
            if (rules.length > 0 && rules[rules.length - 1].Index !== rule.Index) {
              setRules(rules.concat(rule))
            }
          }}
          removeRule={removeRule}
          />
        ) : null}
      </Modal>
    </>
  )
}
