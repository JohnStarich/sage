import axios from 'axios';
import React from 'react';
import Crumb from './Breadcrumb';
import BootstrapTable from 'react-bootstrap-table-next';
import Badge from 'react-bootstrap/Badge';
import './Categories.css';
import { cleanCategory } from './CategoryPicker';

const columns = [
  {
    dataField: 'ID',
    text: 'ID',
    hidden: true,
  },
  {
    dataField: 'Conditions',
    text: 'Conditions',
    formatter: (_, row) => row.Conditions && row.Conditions.map(
      (cond, i) => <Badge key={i} pill variant="light">{cond}</Badge>),
  },
  {
    dataField: 'Account2',
    text: 'Category',
    formatter: cell => {
      let accountType, category;
      if (cell.includes(':')) {
        accountType =  cell.slice(0, cell.indexOf(':'))
        category = cleanCategory(cell)
      } else {
        accountType = cell
        category = ""
      }

      let variant = 'light';
      if (accountType === 'expenses') {
        variant = 'info'
      } else if (accountType === 'revenues') {
        variant = 'success'
      }
      return (
        <div className="category">
          <Badge pill variant={variant}>{accountType}</Badge>
          <span className={category === "uncategorized" ? "account-uncategorized" : null}>{category}</span>
        </div>
      )
    },
  },
];

export default function Categories({ match }) {
  const [rules, setRules] = React.useState([])
  
  React.useEffect(() => {
    axios.get('/api/v1/rules').then(res => {
      let newRules = res.data.Rules || []
      newRules = newRules.map((rule, i) => {
        rule.ID = i
        return rule
      })
      setRules(newRules)
    })
  }, [])

  const handleTableChange = () => {
  }

  return (
    <>
      <Crumb title="Categories" match={match} />
      <BootstrapTable
        keyField="ID"
        data={ rules }
        columns={ columns }

        bootstrap4
        bordered={false}
        noDataIndication="No rules found"
        onTableChange={ handleTableChange }
        remote
        wrapperClasses='table-responsive categories'
        />
    </>
  )
}
