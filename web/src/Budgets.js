import './Budgets.css';
import Amount from './Amount';
import Button from 'react-bootstrap/Button';
import Crumb from './Breadcrumb';
import React from 'react';
import UTCDatePicker from './UTCDatePicker';
import axios from 'axios';
import { cleanCategory, CategoryPicker } from './CategoryPicker';


function parseBudget(budget) {
  return Object.assign({}, budget, {
    Description: cleanCategory(budget.Account),
    Amount: Number(budget.Amount),
    Budget: Number(budget.Budget),
  })
}

function firstAccountComponent(account) {
  const i = account.indexOf(':')
  if (i === -1) {
    return account
  }
  return account.slice(0, i)
}

function sortBudgets(a, b) {
  const aPrefix = firstAccountComponent(a.Account)
  const bPrefix = firstAccountComponent(b.Account)
  const prefixCompare = aPrefix.localeCompare(bPrefix)
  if (prefixCompare !== 0) {
    // if prefixes are different, then:
    if (aPrefix === 'builtin' || bPrefix === 'builtin') {
      // sort "builtin" to the bottom
      return aPrefix === 'builtin' ? 1 : -1
    }
    if (aPrefix === 'revenues' || bPrefix === 'revenues') {
      // sort "revenues" to the top
      return aPrefix === 'revenues' ? -1 : 1
    }
    // sort other prefixes normally
    return prefixCompare
  }
  const compare = a.Description.localeCompare(b.Description)
  if (compare === 0) {
    // sort by full account name if descriptions are equal
    return a.Account.localeCompare(b.Account)
  }
  if (a.Account === a.Description || b.Account === b.Description) {
    // sort "Revenues" or "Expenses" above the other accounts with those prefixes
    return a.Account === a.Description ? -1 : 1
  }
  // otherwise sort by account short name
  return compare
}

function firstOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1))
}

function lastOfMonth(date) {
  return new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth() + 1, 0))
}

function fetchEverythingElseDetails(start, end) {
  return axios.get('/api/v1/getEverythingElseBudget', { params: { start, end } })
    .then(res => {
      const accounts = Object.entries(res.data.Accounts)
        .map(([Account, balance]) => {
          return { Account, Amount: Number(balance), Description: cleanCategory(Account) }
        })
        .sort(sortBudgets)
      return {
        Amount: Number(res.data.Amount),
        Accounts: accounts,
      }
    })
}

export default function Budgets({ match }) {
  const [budgets, setBudgets] = React.useState(null)
  const [timeProgress, setTimeProgress] = React.useState(null)
  const [start, setStart] = React.useState(firstOfMonth(new Date()))
  const [end, setEnd] = React.useState(lastOfMonth(new Date()))
  const [everythingElse, setEverythingElse] = React.useState(null)

  const [addCategory, setAddCategory] = React.useState(null)

  React.useEffect(() => {
    Promise.all([
      axios.get('/api/v1/getBudgets', { params: { start, end } })
        .then(res => res.data.Budgets),
      fetchEverythingElseDetails(start, end),
    ]).then(([budgets, everythingElseDetails]) => {
      setBudgets(budgets.map(parseBudget).sort(sortBudgets))
      const now = new Date()
      const progress = (now.getTime() - start.getTime()) / (end.getTime() - start.getTime())
      setTimeProgress(Math.min(1, progress))
      setEverythingElse(everythingElseDetails)
    })
  }, [start, end])

  if (budgets === null) {
    return <em>Loading...</em>
  }

  const addBudget = (account, budget) => {
    const b = {
      Description: cleanCategory(account),
      Account: account,
      Budget: budget,
    }
    axios.post('/api/v1/addBudget', b)
      .then(() => {
        // fetch current amount before displaying budget
        // also fetch updated "everything else" budget
        Promise.all([
          axios.get('/api/v1/getBudget', { params: { account, start, end } })
            .then(res => res.data.Budget),
          fetchEverythingElseDetails(start, end),
        ]).then(([budget, everythingElseDetails]) => {
          const newBudgets = budgets.slice()
          const newBudget = parseBudget(budget)
          newBudgets.push(newBudget)
          newBudgets.sort(sortBudgets)
          setBudgets(newBudgets)
          setAddCategory(null)
          setEverythingElse(everythingElseDetails)
        })
      })
  }

  const updateBudget = (account, budgetAmount) => {
    if (budgetAmount < 0) {
      budgetAmount = 0
    }
    const existingBudget = budgets.find(b => b.Account === account)
    if (!existingBudget) {
      throw Error(`Budget not found with name: ${account}`)
    }

    const budget = Object.assign({}, existingBudget, {
      Budget: budgetAmount,
    })
    axios.post('/api/v1/updateBudget', budget)
      .then(() => {
        const newBudgets = budgets.filter(b => b.Account !== account)
        newBudgets.push(budget)
        newBudgets.sort(sortBudgets)
        setBudgets(newBudgets)
      })
  }

  const removeBudget = account => {
    if (window.confirm(`Are you sure you want to delete this account? ${account}`)) {
      Promise.all([
        axios.get('/api/v1/deleteBudget', { params: { account } }),
        fetchEverythingElseDetails(start, end),
      ]).then(([_, everythingElseDetails]) => {
        setBudgets(budgets.filter(b => b.Account !== account))
        setEverythingElse(everythingElseDetails)
      })
    }
  }

  return (
    <div className="budgets">
      <Crumb title="Budgets" match={match} />
      <div className="budget-add">
        <CategoryPicker
          category={addCategory}
          setCategory={setAddCategory}
          filter={c => !budgets.find(b => b.Account === c)}
        />
        <Button onClick={() => addBudget(addCategory, 0)} disabled={addCategory === null}>Add budget</Button>
      </div>
      <h2>
        <UTCDatePicker
          dateFormat="MMM yyyy"
          selected={start}
          onChange={v => {
            setStart(firstOfMonth(v))
            setEnd(lastOfMonth(v))
          }}
          maxDate={lastOfMonth(new Date())}
          showMonthYearPicker
        />
      </h2>
      {budgets.map(budget =>
        <Budget
          key={budget.Account}
          name={budget.Description}
          account={budget.Account}
          amount={budget.Account === "builtin:everything else" && everythingElse ? everythingElse.Amount : budget.Amount}
          budget={budget.Budget}
          setBudget={a => updateBudget(budget.Account, a)}

          details={budget.Account === "builtin:everything else" && everythingElse ? everythingElse.Accounts : null}
          timeProgress={timeProgress}
          addBudget={addBudget}
          removeBudget={() => removeBudget(budget.Account)}
        />
      )}
    </div>
  )
}

function Budget({
  name,
  account,
  amount: externalAmount,
  budget: externalBudget,
  setBudget: setExternalBudget,

  details: externalDetails,
  addBudget,
  removeBudget,
  timeProgress,
}) {
  const [budget, setBudget] = React.useState(externalBudget)
  React.useEffect(() => {
    setBudget(externalBudget)
  }, [externalBudget])
  const [amount, setAmount] = React.useState(externalAmount)
  React.useEffect(() => {
    setAmount(externalAmount)
  }, [externalAmount])
  const [details, setDetails] = React.useState(externalDetails)
  React.useEffect(() => {
    setDetails(externalDetails)
  }, [externalDetails])

  const percentage = amount === 0 ? 0 : Math.min(1, amount / budget)
  let budgetColor
  if (amount > budget) {
    budgetColor = "exceeded-budget"
  } else if (percentage - 0.02 > timeProgress) {
    budgetColor = "over-budget"
  } else {
    budgetColor = "on-budget"
  }
  if (percentage > timeProgress) {
    budgetColor += " over-progress"
  }

  let budgetClass = "budget"
  if (account.startsWith("revenues:") || account === 'revenues') {
    budgetClass += " budget-revenues"
  }
  return (
    <div className={budgetClass}>
      <div className="budget-header">
        <div className="budget-name">
          <h5>{name}</h5>
          <h6>{account}</h6>
        </div>
        <div className="budget-controls">
          <div className="budget-amount">
            <Button className="budget-decrease" variant="outline-secondary" onClick={() => setExternalBudget(budget - deltaForIncrement(budget - 1))}>–</Button>
            <Amount prefix="$" amount={budget} onChange={setExternalBudget} editable />
            <Button className="budget-increase" variant="outline-secondary" onClick={() => setExternalBudget(budget + deltaForIncrement(budget))}>+</Button>
          </div>
          {account !== 'builtin:everything else' ?
            <Button className="budget-delete" variant="outline-danger" onClick={removeBudget}>x</Button>
            : null}
        </div>
      </div>
      <div className="budget-graph">
        <div className={"budget-bar " + budgetColor}>
          <div className="budget-filled" style={{ width: `${percentage * 100}%` }} />
          <div className="budget-progress" style={{ width: `${timeProgress * 100}%` }} />
        </div>
        <Amount prefix="$" amount={amount} />
      </div>
      {details ?
        <ul className="budget-details">
          {details.map(budget =>
            <li key={budget.Account}>
              {budget.Description}
              <div>
                <Amount prefix="$" amount={budget.Amount} />
                <Button
                  variant="outline-secondary"
                  onClick={() => {
                    addBudget(budget.Account, budget.Amount)
                  }}
                >+</Button>
              </div>
            </li>
          )}
        </ul>
        : null}
    </div>
  )
}

function deltaForIncrement(amount) {
  if (amount < 100) {
    return 10
  }
  return 100
}
