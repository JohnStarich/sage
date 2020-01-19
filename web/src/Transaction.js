import React from 'react';

import Amount from './Amount';
import Button from 'react-bootstrap/Button';
import Form from 'react-bootstrap/Form';
import Table from 'react-bootstrap/Table';
import { CategoryPicker } from './CategoryPicker';


export default function Transaction(updateTransaction, accountIDMap, editRule) {
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
              <td><Button onClick={() => editRule(txn)}>Add rule</Button></td>
              <td></td>
            </tr>
          </tbody>
        </Table>
      </Form>
    )
  }
}
