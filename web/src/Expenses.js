import React from 'react';
import API from './API';
import {
  Bar,
  BarChart,
  Legend,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import Amount from './Amount';
import Colors from './Colors';
import './Expenses.css';

export default function Expenses(props) {
  const { syncTime } = props
  const [accounts, setAccounts] = React.useState(null)
  const [start, setStart] = React.useState(null)
  const [end, setEnd] = React.useState(null)

  React.useEffect(() => {
    API.get('/v1/getBalances', {
      params: {
        accountTypes: ['expenses', 'revenues', 'uncategorized'],
      },
    })
      .then(res => {
        if (!res.data.Accounts) {
          return
        }
        setAccounts(res.data.Accounts.map(account => {
          account.Balances = account.Balances.map(Number)
          return account
        }))
        setStart(res.data.Start)
        setEnd(res.data.End)
      })
  }, [syncTime])

  const noData = <div className="no-expenses">No expenses data to display</div>
  if (!accounts) {
    return noData
  }
  let accountsCopy = reduceCategories(accounts)
  accountsCopy = accountsCopy.map(removeCumulative).map(negateBalances)
  accountsCopy = sortAccountsByActivity(accountsCopy)
  let data = convertAccountsToChartData({ start, end, accounts: accountsCopy })
  if (data === null) {
    return noData
  }
  return (
    <div className="expenses">
      <ResponsiveContainer width="100%">
        <BarChart data={data} stackOffset="sign" margin={{ left: 50 }}>
          <Tooltip offset={40} content={AmountTooltip} />
          {accountsCopy.map((a, i) =>
            <Bar key={a.ID} dataKey={a.Account} stackId="1" fill={Colors[i % Colors.length]} />
          )}
          <XAxis dataKey="Date" />
          <YAxis tick={AmountTick} />
          <ReferenceLine y={0} />
          <Legend />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

const AmountTooltip = ({ active, payload, label }) => {
  if (!active) {
    return null
  }
  const revenues = [], expenses = []
  const accounts = payload.filter(account => account.value !== 0)
  for (let account of accounts) {
    if (account.value > 0) {
      revenues.push(account)
    } else {
      expenses.push(account)
    }
  }
  revenues.sort((a, b) => b.value - a.value)
  expenses.sort((a, b) => a.value - b.value)

  const makeEntry = (account, i) =>
    <li key={i} className="entry" style={{ color: account.fill }}>
      <span className="account-name">{account.name}</span>
      <Amount amount={account.value} prefix='$' />
    </li>
  const total = amounts =>
    amounts
      .map(v => v.value)
      .reduce((acc, v) => acc + v)

  return (
    <div className="amount-tooltip">
      <p className="label">{label}</p>
      {revenues.length === 0 ? null :
        <>
          <p className="group">Revenues: <Amount prefix="$" amount={total(revenues)} /></p>
          <ul>{revenues.map(makeEntry)}</ul>
        </>
      }
      {expenses.length === 0 ? null :
        <>
          <p className="group">Expenses: <Amount prefix="$" amount={total(expenses)} /></p>
          <ul>{expenses.map(makeEntry)}</ul>
        </>
      }
      {revenues.length === 0 && expenses.length === 0 ?
        <em className="missing">Zero net expenses and revenues.</em>
      : null}
    </div>
  )
}

const AmountTick = tick => {
  let payload = tick.payload;
  // copy nearly all of original tick attributes
  // filter out any invalid attributes
  tick = Object.assign({}, tick);
  [
    'verticalAnchor',
    'visibleTicksCount',
    'payload',
  ].forEach(k => delete tick[k])
  return (
    <text {...tick}>
      <Amount prefix='$' amount={payload.value} tagName='tspan' />
    </text>
  )
}

const dateFormatter = new Intl.DateTimeFormat('default', { year: 'numeric', month: 'long', timeZone: 'UTC' })

function convertAccountsToChartData({ start, end, accounts }) {
  if (!accounts || !end || !start) {
    return null
  }
  if (end <= start) {
    return null
  }
  // Remove trailing Z, since we are only interested in the year and month. Time zone conversions will muddy the water
  start = new Date(start)
  end = new Date(end)
  if (accounts.length === 0) {
    throw new Error("Attempted to convert an empty list of accounts to chart data")
  }
  let times = []

  let year = start.getUTCFullYear()
  let month = start.getUTCMonth()
  let currentDate = new Date(Date.UTC(year, month, 1))
  while (currentDate < end) {
    times.push(currentDate)
    if (month === 11) {
      year++
    }
    month = (month + 1) % 12
    currentDate = new Date(year, month, 1)
  }

  // convert from series of balances and times into large data point objects
  return times.map((time, i) =>
    accounts.reduce((accumulator, account) => {
      accumulator[account.Account] = account.Balances[i]
      return accumulator
    }, { Date: dateFormatter.format(time) })
  )
}

function removeCumulative(account) {
  return Object.assign({}, account, {
    Balances: account.Balances.map((balance, index) => {
      let previousBalance = index === 0 ? 0 : account.Balances[index - 1]
      return balance - previousBalance
    })
  })
}

// negate the balance since expense and revenue accounts are reversed
function negateBalances(account) {
  return Object.assign({}, account, {
    Balances: account.Balances.map(balance => - balance)
  })
}

function reduceCategories(accounts) {
  let accountNames = accounts.map(a => a.Account ? a.Account : a.AccountType)
  accountNames = reduceCategoryNames(accountNames)
  let newAccounts = {}
  for (let account of accounts) {
    for (let name of accountNames) {
      if (account.Account === name ||
        (account.Account === "" && account.AccountType === name) ||
        account.Account.startsWith(name + ':')) {
        if (newAccounts[name] === undefined) {
          account.Account = name
          newAccounts[name] = Object.assign({}, account)
        } else {
          newAccounts[name].Balances =
            newAccounts[name].Balances
              .map((balance, i) => balance + account.Balances[i])
        }
      }
    }
  }
  return Object.values(newAccounts)
}

// try to reduce graph complexity by combining accounts with identical prefixes
function reduceCategoryNames(names) {
  const targetCount = 10

  let allNames = new Set()
  for (let name of names) {
    if (name.includes(':')) {
      allNames.add(name.slice(0, name.indexOf(':')))
    } else {
      allNames.add(name)
    }
  }

  let previousSize = 0
  while (allNames.size < targetCount && previousSize !== allNames.size) {
    previousSize = allNames.size
    for (let name of allNames) {
      let prefix = name + ':'
      let foundPrefixMatch = false
      for (let i = 0; i < names.length && allNames.size < targetCount; i++) {
        if (names[i].startsWith(prefix)) {
          allNames.add(names[i])
          foundPrefixMatch = true
        }
      }
      if (foundPrefixMatch) {
        allNames.delete(name)
        break
      }
    }
  }
  return allNames
}

// assumes current balances are not cumulative
function sortAccountsByActivity(accounts) {
  const getBalance = a =>
    Math.abs(a.Balances.reduce((acc, elem) => acc + elem))
  return accounts.sort((a, b) => getBalance(b) - getBalance(a))
}
