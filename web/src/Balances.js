import React from 'react';
import axios from 'axios';

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
        <ul>
          {Object.entries(this.state.balances)
              .map(bal => {
                let [account, balanceItems] = bal
                let balance = balanceItems[balanceItems.length-1]
                return <li key={account}>{account} -- $ {balance}</li>
              })
          }
        </ul>
      </div>
    )
  }
}
