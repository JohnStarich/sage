import React from 'react';
import axios from 'axios';
import { BarChart, Bar, ResponsiveContainer, Tooltip, XAxis, YAxis, Legend } from 'recharts';
import Amount from './Amount';

const colors = [
  'green',
  'lime',
  'orange',
  'purple',
  'cyan',
  'blue',
  'magenta',
  'violet',
]

export default class Expenses extends React.Component {
  state = {}

  componentDidMount() {
    axios.get('/api/v1/balances', {
        params: {
          accountTypes: ['expenses', 'revenue'],
        },
      })
      .then(res => {
        if (res.status !== 200 ) {
          throw new Error("Error fetching balances")
        }
        this.setState(res.data)
      })
  }

  render() {
    let start = new Date(this.state.Start).getTime()
    let end = new Date(this.state.End).getTime()
    if (! this.state.Accounts || end < start) {
      return <div>No expense data to display</div>
    }
    let interval = (end - start) / this.state.Accounts[0].Balances.length
    let times = []
    for (let i = start; i < end; i += interval) {
      times.push(new Date(i));
    }

    let data = times.map((time, i) =>
      this.state.Accounts.reduce((accumulator, account) => {
        // negate the balance since expense and revenue accounts are reversed
        accumulator[account.Account] = - account.Balances[i]
        return accumulator
      }, { Date: new Date(time).toDateString() })
    )
    return (
      <div style={{ margin: "1em" }}>
        <ResponsiveContainer height={700} width="100%">
          <BarChart title="hi" data={data} stackOffset="sign" margin={{ left: 50, top: 5, right: 5, bottom: 5 }}>
            {this.state.Accounts.map((a, i) =>
              <Bar key={a.ID} dataKey={a.Account} stackId="1" fill={colors[i % colors.length]} />
            )}
            <XAxis dataKey="Date" />
            <YAxis tick={tick => {
              let payload = tick.payload;
              // copy nearly all of original tick attributes
              // filter out any invalid attributes
              [
                'verticalAnchor',
                'visibleTicksCount',
                'payload',
              ].forEach(k => delete tick[k])
              return (
                <text {...tick}>
                  <Amount prefix='$' amount={Number(payload.value)} tagName='tspan' />
                </text>
              )
            }} />
            <Tooltip />
            <Legend />
          </BarChart>
        </ResponsiveContainer>
      </div>
    );
  }
}
