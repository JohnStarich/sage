import React from 'react';
import API from './API';
import './AdvancedOptions.css';

import * as DateUtils from './DateUtils';
import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';
import UTCDatePicker from './UTCDatePicker';
import { Crumb } from './Breadcrumb';
import { Link } from 'react-router-dom';


const labelWidth = 4
const inputWidth = 8

export default function AdvancedOptions({ match }) {
  return (
    <>
      <Crumb title="Advanced Options" match={match} />
      <Container className="advanced-options">
        <Row>
          <h2>Advanced Options</h2>
          <p>These features are dangerous and can corrupt your data. Use wisely.</p>
        </Row>
        <Row>
          <Container>
            <Row>
              <h3>Categorize Transactions</h3>
              <p>Reprocess all uncategorized transactions in the given date period to automatically set their category.</p>
              <p>The category is determined by a transaction's matching <Link to="/settings/categories">category rules</Link>.</p>
            </Row>
            <Row><ReprocessUncategorized /></Row>
          </Container>
        </Row>
        <Row>
          <Container>
            <Row>
              <h3>Rename Ledger Accounts</h3>
              <p>Renames one set of ledger accounts to a new name. Useful for renaming a category or an old, defunct account.</p>
              <p>Posting prefixes are required, ID prefixes are optional.</p>
              <br />
            </Row>
            <Row><RenameAccount /></Row>
          </Container>
        </Row>
      </Container>
    </>
  )
}

function RenameAccount() {
  const [renameSuggestions, setRenameSuggestions] = React.useState(null)
  const [feedback, setFeedback] = React.useState(null)
  const [validated, setValidated] = React.useState(false)

  React.useEffect(() => {
    API.get('/v1/renameSuggestions')
      .then(res => {
        if (res.data.Suggestions) {
          setRenameSuggestions(res.data.Suggestions)
        }
      })
  }, [validated])

  return (
    <Container>
      <Form
        id="rename-account"
        noValidate
        validated={validated}
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          if (form.checkValidity() !== false) {
            const renameParams = {
              Old: document.getElementById("old-account").value,
              New: document.getElementById("new-account").value,
              OldID: document.getElementById("old-account-id").value,
              NewID: document.getElementById("new-account-id").value,
            }
            if (!window.confirm(`Are you sure you want to rename "${renameParams.Old}*" to "${renameParams.New}*"?`)) {
              return
            }
            API.post('/v1/renameLedgerAccount', renameParams)
              .then(res => {
                const renamedCount = res.data.Renamed
                setFeedback(`Success! Renamed ${renamedCount} postings.`)
                form.reset()
                setValidated(false)
              })
              .catch(err => {
                const errorMessage = err.response && err.response.data && err.response.data.Error
                setFeedback(errorMessage || "An internal server error occurred")
                if (!errorMessage) {
                  console.error(err)
                  throw err
                }
              })
          }
          setValidated(true)
        }}
      >
        <Form.Group controlId="old-account" as={Row}>
          <Form.Label column sm={labelWidth}>Old posting prefix</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control required />
          </Col>
        </Form.Group>
        <Form.Group controlId="new-account" as={Row}>
          <Form.Label column sm={labelWidth}>New posting prefix</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control required />
          </Col>
        </Form.Group>
        <Form.Group controlId="old-account-id" as={Row}>
          <Form.Label column sm={labelWidth}>Old ID prefix</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control />
          </Col>
        </Form.Group>
        <Form.Group controlId="new-account-id" as={Row}>
          <Form.Label column sm={labelWidth}>New ID prefix</Form.Label>
          <Col sm={inputWidth}>
            <Form.Control />
          </Col>
        </Form.Group>
        <Row>
          <Col><Button type="submit">Submit</Button></Col>
          <Col>{feedback}</Col>
        </Row>
        {renameSuggestions ?
          <>
            <Row>
              <h5>Rename Suggestions</h5>
            </Row>
            <Row>
              <Container className="rename-suggestions">
                {renameSuggestions.map((r, i) =>
                  <Row key={i}>
                    <Col sm="4">{r.Old}<br />{r.OldID}</Col>
                    <Col sm="1" className="rename-symbol">&#187;</Col>
                    <Col sm="4">{r.New}<br />{r.NewID}</Col>
                    <Col sm="3" className="rename-button"><Button variant="secondary" onClick={() => {
                      document.getElementById("old-account").value = r.Old
                      document.getElementById("new-account").value = r.New
                      document.getElementById("old-account-id").value = r.OldID
                      document.getElementById("new-account-id").value = r.NewID
                    }}>Rename</Button></Col>
                  </Row>
                )}
              </Container>
            </Row>
          </>
          : null}
      </Form>
    </Container>
  )
}

const uncategorizedAccountNames = [
  "expenses:uncategorized",
  "revenues:uncategorized",
  "uncategorized",
]

function ReprocessUncategorized() {
  const [validated, setValidated] = React.useState(false)
  const [start, setStart] = React.useState(DateUtils.firstOfMonth(new Date()))
  const [end, setEnd] = React.useState(start)
  const [txnCount, setTxnCount] = React.useState(null)
  const [feedback, setFeedback] = React.useState(null)

  React.useEffect(() => {
    API.get('/v1/getTransactions', {params: {
      results: 1,
      accounts: uncategorizedAccountNames,
      end,
      start,
    }}).then(res => setTxnCount(res.data.Count))
  }, [start, end])

  return (
    <Container>
      <Form
        id="reprocess-uncategorized"
        noValidate
        validated={validated}
        onSubmit={e => {
          e.preventDefault()
          e.stopPropagation()
          const form = e.currentTarget
          if (form.checkValidity() !== false) {
            if (!window.confirm(`Automatically recategorize ${txnCount} transaction${txnCount === 1 ? "" : "s"}?`)) {
              return
            }
            API.post('/v1/reimportTransactions', { Start: start, End: end, Accounts: uncategorizedAccountNames })
              .then(res => {
                setFeedback(`Success! Auto-categorized ${res.data.Count} transactions.`)
                form.reset()
                const thisMonth = DateUtils.firstOfMonth(new Date())
                setStart(thisMonth)
                setEnd(thisMonth)
                setValidated(false)
              })
              .catch(err => {
                const errorMessage = err.response && err.response.data && err.response.data.Error
                setFeedback(errorMessage || "An internal server error occurred")
                if (!errorMessage) {
                  console.error(err)
                  throw err
                }
              })
          }
          setValidated(true)
        }}
      >
        <Form.Group controlId="reprocess-start" as={Row}>
          <Form.Label column sm={labelWidth}>Start date</Form.Label>
          <Col sm={inputWidth}>
            <UTCDatePicker
              id="reprocess-start"
              selected={start}
              selectsStart
              startDate={start}
              endDate={end}
              onChange={v => {
                setStart(v)
                document.getElementById('reprocess-end').focus()
              }}
              maxDate={DateUtils.lastOfMonth(new Date())}
            />
          </Col>
        </Form.Group>
        <Form.Group controlId="reprocess-end" as={Row}>
          <Form.Label column sm={labelWidth}>End date</Form.Label>
          <Col sm={inputWidth}>
            <UTCDatePicker
              id="reprocess-end"
              selected={end}
              selectsEnd
              startDate={start}
              endDate={end}
              onChange={v => setEnd(v)}
              maxDate={DateUtils.lastOfMonth(new Date())}
            />
          </Col>
        </Form.Group>
        <Row>
          <Col className="reprocess-uncategorized-submit">
            <Button type="submit" disabled={txnCount === 0}>Submit</Button>
            {txnCount !== null ?
              <div className="reprocess-uncategorized-selected"><em>{txnCount} transaction{txnCount === 1 ? "" : "s"} selected</em></div>
            : null}
          </Col>
          <Col>{feedback}</Col>
        </Row>
      </Form>
    </Container>
  )
}
