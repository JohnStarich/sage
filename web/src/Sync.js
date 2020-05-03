import API from './API';
import React from 'react';
import Button from 'react-bootstrap/Button';
import ButtonGroup from 'react-bootstrap/ButtonGroup';
import Card from 'react-bootstrap/Card';
import Col from 'react-bootstrap/Col';
import Container from 'react-bootstrap/Container';
import FontAwesome from 'react-fontawesome';
import Form from 'react-bootstrap/Form';
import Modal from 'react-bootstrap/Modal';
import ListGroup from 'react-bootstrap/ListGroup';
import Row from 'react-bootstrap/Row';
import Spinner from 'react-bootstrap/Spinner';
import moment from 'moment';

import './Sync.css';

async function getStatus() {
  try {
    return (await API.get('/v1/getLedgerSyncStatus')).data
  } catch (e) {
    console.error(e)
    return { Syncing: false, Error: e.message }
  }
}

function submitCode(code) {
  return API.post('/v1/submitSyncPrompt', { Text: code })
}

export default function Sync({ className, onSync }) {
  const [isSyncing, setSyncing] = React.useState(false)
  const [errors, setErrors] = React.useState([])
  const [showModal, setShowModal] = React.useState(false)
  const [prompt, setPrompt] = React.useState(null)

  React.useEffect(() => {
    const checkSync = async () => {
      const status = await getStatus()
      if (onSync && isSyncing && !status.Syncing) {
        onSync()
      }
      setSyncing(status.Syncing)
      setErrors(status.Errors || [])
      setPrompt(status.Prompt || null)
      if (!status.Errors && !status.Prompt) {
        setShowModal(false)
      }
    }

    const interval = setInterval(() => checkSync(), 10000)
    return () => clearInterval(interval)
  }, [isSyncing, onSync])
  React.useEffect(() => {
    getStatus().then(status => {
      setSyncing(status.Syncing)
      setErrors(status.Errors || [])
      setPrompt(status.Prompt || null)
    })
  }, [])

  const clickSync = () => {
    setSyncing(true)
    setErrors([])
    API.post('/v1/syncLedger')
      .catch(e => {
        setSyncing(false)
        setErrors([{ Description: e.message }])
        setPrompt(null)
      })
  }

  let classNames = ["sync"]
  if (className) {
    classNames.push(className)
  }
  if (errors.length > 0) {
    classNames.push("sync-failed")
  }

  let buttonVariant = "dark"
  if (errors.length > 0) {
    buttonVariant = "danger"
  }

  return (
    <div className={classNames.join(' ')}>
      <ButtonGroup>
        <Button
          variant={buttonVariant}
          className="sync-title"
          disabled={isSyncing}
          onClick={!isSyncing ? clickSync : null}
        >
          {isSyncing
            ? <Spinner animation="border" size="sm" />
            : 'Sync'
          }
        </Button>
        {errors.length > 0 || prompt !== null ?
          <Button
            variant={buttonVariant}
            className="sync-info"
            onClick={() => setShowModal(true)}
          >
            <FontAwesome name="exclamation-triangle" />
          </Button>
          : null}
      </ButtonGroup>
      <Modal show={showModal} onHide={() => setShowModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Sync error</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <SyncErrors prompt={prompt} errors={errors} submitPrompt={val => {
            submitCode(val)
            setPrompt(null)
            setShowModal(errors.length > 0)
          }} />
        </Modal.Body>
      </Modal>
    </div>
  )
}

function SyncErrors({ prompt, submitPrompt, errors }) {
  const [code, setCode] = React.useState("")

  if (errors.length === 0 && !prompt) {
    return null
  }

  const groupedErrors = errors && errors.reduce((acc, err) => {
    const title = err.Accounts.join(", ")
    if (!acc[title]) {
      acc[title] = []
    }
    acc[title].push(err)
    return acc
  }, {})

  return (
    <div className="sync-errors">
      {prompt ?
        <Card>
          <ListGroup variant="flush">
            <ListGroup.Item>
              <Form onSubmit={e => {
                e.preventDefault()
                submitPrompt(code)
              }}>
                <Container>
                  <Form.Group as={Row}>
                    <Form.Label column sm="4">{prompt.Message || ""}</Form.Label>
                    <Col>
                      <Form.Control sm="8" type="text" value={code} onChange={e => {
                        setCode(e.target.value)
                      }} />
                    </Col>
                  </Form.Group>
                  <Row>
                    <Col><Button onClick={() => submitPrompt(code)}>Submit</Button></Col>
                  </Row>
                </Container>
              </Form>
            </ListGroup.Item>
          </ListGroup>
        </Card>
        : null}
      {groupedErrors ? Object.entries(groupedErrors)
        .map(([title, errs], i) => (
          <Card key={i}>
            <Card.Header>{title}</Card.Header>
            <ListGroup variant="flush">
              {errs.map((err, i) =>
                <div key={i}>
                  {err.Records ?
                    err.Records.map((rec, i) =>
                      <div key={i}>
                        <div className="sync-record-time"><em>{moment(rec.CreatedTime).fromNow()}</em></div>
                        <RecordCard record={rec} />
                      </div>
                    )
                    : null}
                  <ListGroup.Item>
                    <pre className="sync-error"><code>{err.Description}</code></pre>
                  </ListGroup.Item>
                </div>
              )}
            </ListGroup>
          </Card>
        ))
        : null}
    </div>
  )
}

function RecordCard({ record }) {
  const { ContentType, Data } = record
  switch (ContentType) {
    case "image/gif":
      return <Card.Img src={`data:${ContentType};base64,${Data}`} alt="Web Connect screen recording for error" />
    case "text/plain":
      return <Card.Text>
        <pre className="sync-error"><code>{window.atob(Data)}</code></pre>
      </Card.Text>
    default:
      return <Card.Text>Unrecognized content type: {ContentType}</Card.Text>
  }
}
