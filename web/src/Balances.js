import React from 'react';
import axios from 'axios';
import './Balances.css';
import Amount from './Amount';

export default class Balances extends React.Component {
  state = {
    balances: {},
    start: "",
    end: "",
  }

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
        <table>
          <tbody>
            <tr><th>Account</th><th>Balance</th></tr>
            {Object.entries(this.state.balances)
                .filter(bal => {
                  let fullAccount = bal[0]
                  return fullAccount.startsWith("assets:") || fullAccount.startsWith("liabilities:")
                })
                .map(bal => {
                  let [fullAccount, balanceItems] = bal

                  let className = ""
                  let balance = Number(balanceItems[balanceItems.length-1])
                  let accountType = ""
                  let account = fullAccount
                  let firstColon = account.indexOf(':')
                  if (firstColon > 0) {
                    accountType = account.slice(0, firstColon)
                    account = account.slice(firstColon + 1)
                    className += " account-" + accountType
                  }
                  account = account.replace(":", " - ")
                  return (
                    <tr key={fullAccount}>
                      <td className={className}>{account}</td>
                      <td><Amount amount={balance} prefix='$' /></td>
                    </tr>
                  )
                })
            }
          </tbody>
        </table>
      </div>
    )
  }
}
