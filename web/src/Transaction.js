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
                <TransactionRules transaction={txn} />
              </td>
            </tr>
          </tbody>
        </Table>
      </Form>
    )
  }
}

function TransactionRules({ transaction }) {
  const [editing, setEditing] = React.useState(null)
  const [rule, setRule] = React.useState(null)
  React.useEffect(() => {
    if (!transaction) {
      return
    }
    API.get('/v1/getRules', { params: { transaction: transaction.ID } })
      .then(res => {
        const rules = Object.entries(res.data.Rules)
          .map(([key, value]) =>
            Object.assign({ Index: key }, value)
          )
          .filter(r => r.Conditions && r.Conditions.length > 0)
          .sort((a, b) => a.Index - b.Index)
        if (rules && rules.length > 0) {
          const newRule = rules[rules.length - 1]
          newRule.Index = Number(newRule.Index)
          setRule(newRule)
        }
      })
  }, [transaction])

  const account2 = transaction.Postings[1].Account
  const isUncategorized = account2 === 'uncategorized' || account2 === 'expenses:uncategorized'

  return (
    <>
      {!isUncategorized ?
        <Button
          className="edit-rule-btn"
          onClick={() => setEditing(rule || {Conditions: [], Account2: account2})}
          variant={rule ? "secondary" : "link"}
          >
          {rule
            ? "Edit rule"
            : <>Always categorize "{transaction.Payee}" as <strong>{cleanCategory(transaction.Postings[1].Account)}</strong>?</>
          }
        </Button>
      : null}
      <Modal show={editing !== null} onHide={() => setEditing(null)}>
        {editing !== null ? (
        <RuleEditor
          onClose={() => setEditing(null)}
          transaction={transaction}
          rule={editing}
          setRule={setRule}
          />
        ) : null}
      </Modal>
    </>
  )
}
