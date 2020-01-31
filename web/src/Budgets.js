import './Budgets.css';
import * as DateUtils from './DateUtils';
import API from './API';
import Amount from './Amount';
import BudgetsHistory from './BudgetsHistory';
import Button from 'react-bootstrap/Button';
import Crumb from './Breadcrumb';
import React from 'react';
import UTCDatePicker from './UTCDatePicker';
import { cleanCategory, CategoryPicker } from './CategoryPicker';


function parseBudget(budget) {
  return Object.assign({}, budget, {
    Description: cleanCategory(budget.Account),
    Balance: Number(budget.Balance),
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

function fetchEverythingElseDetails(start, end) {
  return API.get('/v1/getEverythingElseBudget', { params: { start, end } })
    .then(res => {
      const accounts = Object.entries(res.data.Accounts)
        .map(([Account, balance]) => {
          return { Account, Balance: Number(balance), Description: cleanCategory(Account) }
        })
        .sort(sortBudgets)
      return {
        Balance: Number(res.data.Balance),
        Accounts: accounts,
      }
    })
}

function parseAllBudgets(budgets) {
  return budgets.map(accounts => accounts.map(parseBudget).sort(sortBudgets))
}

export default function Budgets({ match }) {
  const [allBudgets, setAllBudgets] = React.useState(null)
  const [budgets, setBudgets] = React.useState(null)
  const [timeProgress, setTimeProgress] = React.useState(null)
  const [start, setStart] = React.useState(DateUtils.firstOfMonth(new Date()))
  const [end, setEnd] = React.useState(DateUtils.lastOfMonth(new Date()))
  const [everythingElse, setEverythingElse] = React.useState(null)
  const [controlsEnabled, setControlsEnabled] = React.useState(false)

  const [addCategory, setAddCategory] = React.useState(null)

  React.useEffect(() => {
    Promise.all([
      API.get('/v1/getBudgets', { params: { start, end } })
        .then(res => res.data.Budgets),
      fetchEverythingElseDetails(start, end),
    ]).then(([budgets, everythingElseDetails]) => {
      const all = parseAllBudgets(budgets)
      setBudgets(all[all.length-1])
      const now = new Date()
      const progress = (now.getTime() - start.getTime()) / (end.getTime() - start.getTime())
      setTimeProgress(Math.min(1, progress))
      setEverythingElse(everythingElseDetails)
      setControlsEnabled(DateUtils.firstOfMonth(now).getTime() === start.getTime() && DateUtils.lastOfMonth(now).getTime() === end.getTime())
    })
  }, [start, end])

  React.useEffect(() => {
    const now = new Date()
    const start = DateUtils.someMonthsAgo(now, 12), end = DateUtils.lastOfMonth(now)
    API.get('/v1/getBudgets', { params: { start, end } })
      .then(res => {
        setAllBudgets(Object.assign(
          { Budgets: parseAllBudgets(res.data.Budgets) },
          res.data,
        ))
      })
  }, [])

  if (budgets === null) {
    return <em>Loading...</em>
  }

  const addBudget = (account, budget) => {
    const b = {
      Description: cleanCategory(account),
      Account: account,
      Budget: budget,
    }
    API.post('/v1/updateBudget', b)
      .then(() => {
        // fetch current balance before displaying budget
        // also fetch updated "everything else" budget
        Promise.all([
          API.get('/v1/getBudget', { params: { account, start, end } })
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

  const updateBudget = (account, budgetBalance) => {
    if (budgetBalance < 0) {
      budgetBalance = 0
    }
    const existingBudget = budgets.find(b => b.Account === account)
    if (!existingBudget) {
      throw Error(`Budget not found with name: ${account}`)
    }

    const budget = Object.assign({}, existingBudget, {
      Budget: budgetBalance,
    })
    API.post('/v1/updateBudget', budget)
      .then(() => {
        const newBudgets = budgets.filter(b => b.Account !== account)
        newBudgets.push(budget)
        newBudgets.sort(sortBudgets)
        setBudgets(newBudgets)
      })
  }

  const removeBudget = budget => {
    if (window.confirm(`Are you sure you want to delete this budget? ${budget}`)) {
      Promise.all([
        API.get('/v1/deleteBudget', { params: { budget } }),
        fetchEverythingElseDetails(start, end),
      ]).then(([_, everythingElseDetails]) => {
        setBudgets(budgets.filter(b => b.Account !== budget))
        setEverythingElse(everythingElseDetails)
      })
    }
  }

  return (
    <div className="budgets">
      <Crumb title="Budgets" match={match} />
        <BudgetsHistory
          budgets={allBudgets}
          date={start}
          setMonth={start => {
            setStart(start)
            setEnd(DateUtils.lastOfMonth(start))
          }}
          />
      <h2>
        <UTCDatePicker
          dateFormat="MMM yyyy"
          selected={start}
          onChange={v => {
            setStart(DateUtils.firstOfMonth(v))
            setEnd(DateUtils.lastOfMonth(v))
          }}
          maxDate={DateUtils.lastOfMonth(new Date())}
          showMonthYearPicker
        />
      </h2>
      <div className="budget-add">
        <CategoryPicker
          category={addCategory}
          setCategory={setAddCategory}
          filter={c => !budgets.find(b => b.Account === c)}
        />
        <Button
          onClick={() => addBudget(addCategory, 0)}
          disabled={addCategory === null || !controlsEnabled}
          >Add budget</Button>
      </div>
      {budgets.map(budget =>
        <Budget
          key={budget.Account}
          name={budget.Description}
          account={budget.Account}
          balance={budget.Account === "builtin:everything else" && everythingElse ? everythingElse.Balance : budget.Balance}
          budget={budget.Budget}
          setBudget={a => updateBudget(budget.Account, a)}

          details={budget.Account === "builtin:everything else" && everythingElse ? everythingElse.Accounts : null}
          disabled={!controlsEnabled}
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
  balance: externalBalance,
  budget: externalBudget,
  setBudget: setExternalBudget,

  disabled = false,
  details: externalDetails,
  addBudget,
  removeBudget,
  timeProgress,
}) {
  const [budget, setBudget] = React.useState(externalBudget)
  React.useEffect(() => {
    setBudget(externalBudget)
  }, [externalBudget])
  const [balance, setBalance] = React.useState(externalBalance)
  React.useEffect(() => {
    setBalance(externalBalance)
  }, [externalBalance])
  const [details, setDetails] = React.useState(externalDetails)
  React.useEffect(() => {
    setDetails(externalDetails)
  }, [externalDetails])

  const percentage = balance === 0 ? 0 : Math.min(1, balance / budget)
  let budgetColor
  if (balance > budget) {
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
            <Button className="budget-decrease" variant="outline-secondary" disabled={disabled} onClick={() => setExternalBudget(budget - deltaForIncrement(budget - 1))}>–</Button>
            <Amount prefix="$" amount={budget} disabled={disabled} onChange={setExternalBudget} editable />
            <Button className="budget-increase" variant="outline-secondary" disabled={disabled} onClick={() => setExternalBudget(budget + deltaForIncrement(budget))}>+</Button>
          </div>
          {account !== 'builtin:everything else' ?
            <Button className="budget-delete" variant="outline-danger" disabled={disabled} onClick={removeBudget}>x</Button>
            : null}
        </div>
      </div>
      <div className="budget-graph">
        <div className={"budget-bar " + budgetColor}>
          <div className="budget-filled" style={{ width: `${percentage * 100}%` }} />
          <div className="budget-progress" style={{ width: `${timeProgress * 100}%` }} />
        </div>
        <Amount prefix="$" amount={balance} />
      </div>
      {details ?
        <ul className="budget-details">
          {details.map(budget =>
            <li key={budget.Account}>
              {budget.Description}
              <div>
                <Amount prefix="$" amount={budget.Balance} />
                <Button
                  variant="outline-secondary"
                  disabled={disabled}
                  onClick={() => {
                    addBudget(budget.Account, budget.Balance)
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
