import React from 'react';
import axios from 'axios';
import './Balances.css';
import Amount from './Amount';
import Table from 'react-bootstrap/Table';

export default class Balances extends React.Component {
  state = {}

  componentDidMount() {
    axios.get('/api/v1/balances')
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching balances")
        }
        this.setState(res.data)
      })
  }

  render() {
    return (
      <div className="balances">
        <Table responsive>
          <tbody>
            <tr><th>Account</th><th>Balance</th></tr>
            {this.state.Accounts && this.state.Accounts.map(account =>
                <tr key={account.ID}>
                  <td>{account.Institution} - {account.Account}</td>
                  <td>
                    <Amount
                      amount={Number(account.Balances[account.Balances.length - 1])}
                      highlightNegative
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
}
