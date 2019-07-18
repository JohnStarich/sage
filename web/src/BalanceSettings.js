import axios from 'axios';
import React from 'react';
import Crumb from './Breadcrumb';
import Amount from './Amount';
import BootstrapTable from 'react-bootstrap-table-next';
import cellEditFactory from 'react-bootstrap-table2-editor';
import FunctionalEditor from './FunctionalEditor';
import UTCDatePicker from './UTCDatePicker';
import "react-datepicker/dist/react-datepicker.css";
import Form from 'react-bootstrap/Form';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import './BalanceSettings.css';


const dateFormatter = new Intl.DateTimeFormat('default', {year: 'numeric', month: 'long', day: 'numeric', timeZone: 'UTC'})

function firstDayOfYear() {
  const now = new Date()
  const date = new Date(Date.UTC(now.getUTCFullYear(), 0, 1))
  return date
}

export default function BalanceSettings({ match }) {
  const [start, setStart] = React.useState(null)
  const [postings, setPostings] = React.useState([])

  React.useEffect(() => {
    axios.get('/api/v1/balances')
      .then(res => {
        window.setTimeout(() => {
        if (res.data.Accounts) {
          setPostings(res.data.Accounts
            .map(({ ID: Account, Account: Description, OpeningBalance: Amount }) => {
              return { Account, Description, Amount }
            }))
        }
        setStart(
          res.data.OpeningBalanceDate
          ? new Date(res.data.OpeningBalanceDate)
          : firstDayOfYear()
        )
      }, 1000)
      })
  }, [])

  const updateOpeningBalance = start => {
    if (!start || !postings || postings.length === 0) {
      return
    }

    axios.put('/api/v1/balances/opening', { Postings: postings, Date: start })
      .catch(e => alert(e))
  }

  const cellEdit = cellEditFactory({
    mode: 'click',
    blurToSave: true,
    afterSaveCell: (oldValue, newValue) => {
      if (oldValue === newValue) {
        return
      }
      updateOpeningBalance(start)
    },
  });

  const columns = [
    {
      dataField: 'Description',
      text: 'Account',
      editable: false,
    },
    {
      dataField: 'Amount',
      text: start ? `Balance as of ${dateFormatter.format(start)}` : "Balance",
      align: 'right',
      headerAlign: 'right',
      formatter: amount => amount ? <Amount amount={Number(amount)} prefix="$" /> : <em>Click to edit...</em>,
      editorRenderer: (props, value) => {
        return (
          <FunctionalEditor {...props}>
            <Amount
              amount={Number(value)}
              prefix="$"
              editable
              autoFocus
              />
          </FunctionalEditor>
        )
      },
    },
  ];

  return (
    <Container className="balance-settings">
      <Crumb title="Balances" match={match} />
      <Row>
        <Col><h2>Balances</h2></Col>
      </Row>
      <Form.Group controlId="start-date" as={Row}>
        <Form.Label column>Start date</Form.Label>
        <Col>
          <UTCDatePicker
            customInput={<Form.Control />}
            selected={start}
            onChange={v => {
              setStart(v)
              updateOpeningBalance(v)
            }}
            popperPlacement="top"
            />
        </Col>
      </Form.Group>
      <Row>
        <Col><p>Click a balance cell to update an opening balance.</p></Col>
      </Row>
      <Row>
        <BootstrapTable
          keyField="Account"
          data={ postings }
          columns={ columns }

          bootstrap4
          bordered={ false }
          cellEdit={ cellEdit }
          noDataIndication="No accounts found"
          wrapperClasses='table-responsive'
          />
      </Row>
    </Container>
  )
}
