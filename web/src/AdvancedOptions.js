import React from 'react';
import API from './API';
import './AdvancedOptions.css';

import Button from 'react-bootstrap/Button';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import Form from 'react-bootstrap/Form';
import Row from 'react-bootstrap/Row';
import { Crumb } from './Breadcrumb';


const labelWidth = 4
const inputWidth = 8

export default function({ match }) {
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
            if (! window.confirm(`Are you sure you want to rename "${renameParams.Old}*" to "${renameParams.New}*"?`)) {
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
                if (! errorMessage) {
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
                  <Col sm="3" className="rename-button"><Button variant="secondary" onClick={e => {
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
