import Badge from 'react-bootstrap/Badge';
import BootstrapTable from 'react-bootstrap-table-next';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Crumb from './Breadcrumb';
import React from 'react';
import Row from 'react-bootstrap/Row';
import axios from 'axios';
import './Categories.css';
import cellEditFactory, { Type } from 'react-bootstrap-table2-editor';
import { cleanCategory } from './CategoryPicker';


const columns = [
  {
    dataField: 'ID',
    text: 'Rule #',
    headerClasses: 'categories-small-width',
    align: 'right',
    editable: () => true,
  },
  {
    dataField: 'Conditions',
    text: 'Conditions',
    formatter: cell =>
      cell.split('\n')
        .map((cond, i) =>
          <Badge key={i} pill variant="light">{cond}</Badge>),
    editor: {
      type: Type.TEXTAREA,
      placeholder: "paycheck from company\nwaffle house\nexxonmobil",
      rows: 5,
    },
  },
  {
    dataField: 'Account2',
    text: 'Category',
    validator: (newValue, row) => {
      const [accountType] = newValue.split(':', 1)
      if (newValue === "" && row.Conditions === "") {
        // special case to delete a row
        return true
      }
      if (accountType !== 'expenses' && accountType !== 'revenues') {
        return { valid: false, message: 'Category must start with "expenses:" or "revenues:"' }
      }
      return true
    },
    editor: {
      placeholder: 'expenses:Shopping:Food:Restaurants',
    },
    formatter: cell => {
      if (!cell) {
        return null
      }

      let accountType, category;
      if (cell.includes(':')) {
        accountType = cell.slice(0, cell.indexOf(':'))
        category = cleanCategory(cell)
      } else {
        accountType = cell
        category = ""
      }

      let variant = 'secondary';
      if (accountType === 'expenses') {
        variant = 'info'
      } else if (accountType === 'revenues') {
        variant = 'success'
      }

      let accountClass = category === 'uncategorized' ? 'account-uncategorized' : null
      return (
        <div className="category">
          <Badge pill variant={variant}>{accountType}</Badge>
          <span className={accountClass}>{category}</span>
        </div>
      )
    },
  },
];

export default function Categories({ match }) {
  const [rules, setRules] = React.useState([])

  React.useEffect(() => {
    axios.get('/api/v1/getRules').then(res => {
      let newRules = res.data.Rules || []
      newRules = newRules.map((rule, i) => {
        rule.ID = i + 1
        rule.Conditions = rule.Conditions.join('\n')
        return rule
      })
      setRules(newRules)
    })
  }, [])

  const updateRules = newRules => {
    const apiRules = newRules.map(rule => Object.assign({}, rule, {
      ID: undefined,
      Conditions: rule.Conditions.split('\n'),
    }))
    axios.post('/api/v1/updateRules', apiRules)
      .then(() => setRules(newRules))
      .catch(e => alert(`Error saving rules. ${e.response.data.Error || ""}`))
  }

  const cellEdit = cellEditFactory({
    mode: 'click',
    blurToSave: true,
    afterSaveCell: (oldValue, newValue, row, column) => {
      if (oldValue.toString() === newValue.toString()) {
        return
      }
      if (column.dataField === 'ID') {
        // updated row order
        const oldIndex = Number(oldValue) - 1
        let newIndex = Number(newValue) - 1
        if (newIndex < 0) {
          newIndex = 0
        }
        let oldRule = rules[oldIndex]
        // pop off old rule
        let newRules =
          rules.slice(0, oldIndex)
            .concat(rules.slice(oldIndex + 1))
        // re-insert old rule at it's new index
        newRules =
          newRules.slice(0, newIndex)
            .concat([oldRule])
            .concat(newRules.slice(newIndex))
        // re-number every rule
        newRules = renumberRules(newRules)
        updateRules(newRules)
        return
      }

      if (row.Conditions === "" && row.Account2 === "") {
        // if all fields are zeroed out, delete the row
        const index = Number(row.ID) - 1
        const newRules = rules.slice(0, index).concat(rules.slice(index + 1))
        updateRules(renumberRules(newRules))
        return
      }

      // normal update
      updateRules(rules)
    },
  })

  const addRule = () => {
    const newRule = {
      Conditions: 'food',
      Account2: 'expenses:shopping:food and restaurants',
    }
    const newRules = renumberRules([newRule].concat(rules))
    updateRules(newRules)
  }

  /*
  // TODO use this instead of special-case editing removal
  const removeRule = i => {
    updateRules(rules.slice(0, i).concat(rules.slice(i + 1)))
  }
  */

  return (
    <>
      <Crumb title="Categories" match={match} />
      <Container>
        <Row>
          <Col><h2>Categories</h2></Col>
        </Row>
        <Row>
          <Col>
            <p>
              Add rules to auto-categorize new transactions. Every condition that matches a transaction will set the category, the last rule wins.
            </p>
            <p>
              Click to edit the rule fields.
              Re-order rules by editing their rule number.
              Each condition goes on its own line.
            </p>
            <Button variant="primary" onClick={addRule}>Add</Button>
          </Col>
        </Row>
      </Container>
      <BootstrapTable
        keyField="ID"
        data={rules}
        columns={columns}

        bootstrap4
        bordered={false}
        cellEdit={cellEdit}
        className="categories-table"
        noDataIndication="No rules found"
        wrapperClasses='table-responsive categories'
      />
    </>
  )
}

function renumberRules(rules) {
  return rules.map((rule, i) => Object.assign({}, rule, { ID: i + 1 }))
}
