import React from 'react';

import Amount from './Amount';
import Form from 'react-bootstrap/Form';
import Table from 'react-bootstrap/Table';
import { CategoryPicker } from './Categories';


export default function Transaction(updateTransaction) {
  return txn => {
    let postings = txn.Postings.map((p, i) => {
      if (i === 0) {
        let account = p.Account;
        let separatorIndex = account.indexOf(':')
        account = separatorIndex !== -1 ? account.slice(separatorIndex + 1) : account
        account = account.replace(/:/, " - ")
        return Object.assign({}, p, { Account: account })
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
                  { i === 0
                    ? <Form.Control type="text" value={posting.Account} disabled />
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
          </tbody>
        </Table>
      </Form>
    )
  }
}
