import React from 'react';
import './BudgetsHistory.css';

import Amount from './Amount';
import Button from 'react-bootstrap/Button';


const monthFormat = new Intl.DateTimeFormat('default', { year: 'numeric', month: 'short', timeZone: 'UTC' });

function addMonths(date, months) {
  const dateCopy = new Date(date)
  dateCopy.setUTCMonth(dateCopy.getUTCMonth() + months)
  return dateCopy
}

export default function BudgetsHistory({ budgets: rawBudgets, date, setMonth }) {
  if (!date) {
    throw Error("date is required")
  }
  if (!setMonth) {
    throw Error("setMonth is required")
  }
  if (!rawBudgets) {
    return null
  }
  const start = new Date(rawBudgets.Start)
  const budgets =
    rawBudgets.Budgets.map(month =>
      month
        .filter(b => !b.Account.startsWith("revenues:") && b.Account !== "revenues")
        .map(b => b.Balance - b.Budget)
        .reduce((a, b) => a + b))
  return (
    <div className="budgets-history">
      {budgets.map((delta, i) => {
        const month = addMonths(start, i)
        const active = date.getTime() === month.getTime()
        return (
          <Button
            key={i}
            className={"budget " + (delta < 0 ? "budget-under" : "budget-over")}
            onClick={() => setMonth(month)}
            active={active}
            variant="light"
            >
            <div className="budget-delta">
              <div className="budget-trend">{delta < 0 ? '▼' : '▲'}</div>
              <Amount prefix="$" amount={Math.abs(delta)} />
            </div>
            <div className="budget-date">{monthFormat.format(month)}</div>
          </Button>
          )
        }
      ).reduce((arr, item, index) => {
        // alternates which month goes where so the column layout appears to be in order by row
        // NOTE: currently fixed at exactly 2 rows
        index *= 2
        if (index >= budgets.length) {
          index = (index + 1) % budgets.length
        }
        arr[index] = item
        return arr
      }, new Array(budgets.length))}
    </div>
  )
}
