import API from './API';
import React from 'react';
import Button from 'react-bootstrap/Button';
import ButtonGroup from 'react-bootstrap/ButtonGroup';
import Card from 'react-bootstrap/Card';
import FontAwesome from 'react-fontawesome';
import Modal from 'react-bootstrap/Modal';
import ListGroup from 'react-bootstrap/ListGroup';
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

export default function Sync({ className, onSync }) {
  const [isSyncing, setSyncing] = React.useState(false)
  const [errors, setErrors] = React.useState(null)
  const [showErrors, setShowErrors] = React.useState(false)

  React.useEffect(() => {
    const checkSync = async () => {
      const status = await getStatus()
      if (onSync && isSyncing && !status.Syncing) {
        onSync()
      }
      setSyncing(status.Syncing)
      setErrors(status.Errors || null)
      if (!status.Errors && showErrors) {
        setShowErrors(false)
      }
    }

    const interval = setInterval(() => checkSync(), 10000)
    return () => clearInterval(interval)
  }, [isSyncing, onSync])
  React.useEffect(() => {
    getStatus().then(status => {
      setSyncing(status.Syncing)
      setErrors(status.Errors || null)
    })
  }, [])

  const clickSync = () => {
    setSyncing(true)
    setErrors(null)
    API.post('/v1/syncLedger')
      .catch(e => {
        setSyncing(false)
        setErrors([{ Description: e.message }])
      })
  }

  let classNames = ["sync"]
  if (className) {
    classNames.push(className)
  }
  if (errors) {
    classNames.push("sync-failed")
  }

  let buttonVariant = "dark"
  if (errors) {
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
        {errors !== null ?
          <Button
            variant={buttonVariant}
            className="sync-info"
            onClick={() => setShowErrors(true)}
          >
            <FontAwesome name="exclamation-triangle" />
          </Button>
          : null}
      </ButtonGroup>
      <Modal show={showErrors} onHide={() => setShowErrors(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Sync error</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <SyncErrors errors={errors} />
        </Modal.Body>
      </Modal>
    </div>
  )
}

function SyncErrors({ errors }) {
  if (!errors) {
    return null
  }

  const groupedErrors = errors.reduce((acc, err) => {
    const title = err.Accounts.join(", ")
    if (!acc[title]) {
      acc[title] = []
    }
    acc[title].push(err)
    return acc
  }, {})

  return (
    <div className="sync-errors">
      {Object.entries(groupedErrors)
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
      }
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
