import React from 'react';
import axios from 'axios';
import './Balances.css';
import Amount from './Amount';
import Table from 'react-bootstrap/Table';

export default function Balances(props) {
  const [payload, setPayload] = React.useState({})
  const { syncTime } = props;

  React.useEffect(() => {
    axios.get('/api/v1/balances')
      .then(res => setPayload(Object.assign({}, res.data)))
  }, [syncTime])

  return (
    <div className="balances">
      <Table responsive>
        <tbody>
          <tr><th>Account</th><th>Balance</th></tr>
          {payload.Accounts && payload.Accounts.map(account =>
              <tr key={account.ID}>
                <td>{account.Account}</td>
                <td>
                  <Amount
                    amount={Number(account.Balances[account.Balances.length - 1])}
                    prefix="$"
                    />
                </td>
              </tr>
          )}
        </tbody>
      </Table>
    </div>
  )
}
