import './Budgets.css';
import React from 'react';
import axios from 'axios';
import Amount from './Amount';
import Button from 'react-bootstrap/Button';
import Crumb from './Breadcrumb';
import { cleanCategory, CategoryPicker } from './CategoryPicker';

function parseBudget(budget) {
  return Object.assign({}, budget, {
    Description: cleanCategory(budget.Account),
    Amount: Number(budget.Amount),
    Budget: Number(budget.Budget),
  })
}

export default function Budgets({ match }) {
  const [budgets, setBudgets] = React.useState(null)
  const [timeProgress, setTimeProgress] = React.useState(null)

  const [addCategory, setAddCategory] = React.useState(null)

  React.useEffect(() => {
    const now = new Date()
    const start = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1))
    const end = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0))
    axios.get('/api/v1/getBudgets', { params: { start, end } })
      .then(res => {
        setBudgets(res.data.Budgets
          .map(parseBudget)
          .sort((a, b) => a.Description.localeCompare(b.Description))
        )
        const progress = (now.getTime() - start.getTime()) / (end.getTime() - start.getTime())
        setTimeProgress(Math.min(1, progress))
      })
  }, [])

  if (budgets === null) {
    return <em>Loading...</em>
  }

  const addBudget = (account, budget) => {
    const b = {
      Description: cleanCategory(account),
      Account: account,
      Amount: 0,
      Budget: budget,
    }
    axios.post('/api/v1/addBudget', b)
      .then(() => {
        // fetch current amount before displaying budget
        axios.get('/api/v1/getBudget', { params: { account } })
          .then(res => {
            const newBudgets = budgets.slice()
            const newBudget = parseBudget(res.data.Budget)
            newBudgets.push(newBudget)
            setBudgets(newBudgets)
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
        newBudgets.sort((a, b) => a.Description.localeCompare(b.Description))
        setBudgets(newBudgets)
      })
  }

  const removeBudget = account => {
    if (window.confirm(`Are you sure you want to delete this account? ${account}`)) {
      axios.get('/api/v1/deleteBudget', { params: { account } })
        .then(() => setBudgets(budgets.filter(b => b.Account !== account)))
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
        <Button onClick={() => addBudget(addCategory, 0)}>Add budget</Button>
      </div>
      {budgets.map((budget, i) =>
        <Budget
          key={i}
          name={budget.Description}
          amount={budget.Amount}
          budget={budget.Budget}
          setBudget={a => updateBudget(budget.Account, a)}
          timeProgress={timeProgress}
          removeBudget={() => removeBudget(budget.Account)}
        />
      )}
    </div>
  )
}

function Budget({
  name,
  amount,
  budget,
  setBudget,
  timeProgress,
  removeBudget,
}) {
  const percentage = Math.min(1, amount / budget)
  let budgetColor = ""
  if (percentage - 0.05 < timeProgress) {
    budgetColor = "below-progress"
  } else {
    budgetColor = "over-progress"
  }
  if (amount > budget) {
    budgetColor += " over-budget"
  }
  return (
    <div className="budget">
      <div className="budget-header">
        <h5 className="budget-name">{name}</h5>
        <div className="budget-controls">
          <div className="budget-amount">
            <Button className="budget-decrease" variant="outline-secondary" onClick={() => setBudget(budget - deltaForIncrement(budget - 1))}>–</Button>
            <Amount key={budget} prefix="$" amount={budget} onChange={setBudget} editable />
            <Button className="budget-increase" variant="outline-secondary" onClick={() => setBudget(budget + deltaForIncrement(budget))}>+</Button>
          </div>
          <Button className="budget-delete" variant="outline-danger" onClick={removeBudget}>x</Button>
        </div>
      </div>
      <div className="budget-graph">
        <div className={"budget-bar " + budgetColor}>
          <div className="budget-filled" style={{ width: `${percentage * 100}%` }} />
          <div className="budget-progress" style={{ width: `${timeProgress * 100}%` }} />
        </div>
        <Amount prefix="$" amount={amount} />
      </div>
    </div>
  )
}

function deltaForIncrement(amount) {
  if (amount < 100) {
    return 10
  }
  return 100
}
