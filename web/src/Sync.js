import API from './API';
import React from 'react';
import Button from 'react-bootstrap/Button';
import ButtonGroup from 'react-bootstrap/ButtonGroup';
import FontAwesome from 'react-fontawesome';
import Modal from 'react-bootstrap/Modal';
import Spinner from 'react-bootstrap/Spinner';

import './Sync.css';

async function getStatus() {
  try {
    return (await API.get('/v1/getLedgerSyncStatus')).data
  } catch (e) {
    console.error(e)
    return
  }
}

export default function Sync({ className, onSync }) {
  const [isSyncing, setSyncing] = React.useState(false)
  const [error, setError] = React.useState(null)
  const [showError, setShowError] = React.useState(false)

  React.useEffect(() => {
    const checkSync = async () => {
      const status = await getStatus()
      if (onSync && isSyncing && !status.Syncing) {
        onSync()
      }
      setSyncing(status.Syncing)
      setError(status.Error || null)
    }

    const interval = setInterval(() => checkSync(), 10000)
    return () => clearInterval(interval)
  }, [isSyncing, onSync])
  React.useEffect(() => {
    getStatus().then(status => {
      setSyncing(status.Syncing)
      setError(status.Error || null)
    })
  }, [])

  const clickSync = () => {
    setSyncing(true)
    setError(null)
    API.post('/v1/syncLedger')
  }

  let classNames = ["sync"]
  if (className) {
    classNames.push(className)
  }
  if (error) {
    classNames.push("sync-failed")
  }

  let buttonVariant = "dark"
  if (error) {
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
        {error ?
          <Button
            variant={buttonVariant}
            className="sync-info"
            onClick={() => setShowError(true)}
          >
            <FontAwesome name="exclamation-triangle" />
          </Button>
          : null}
      </ButtonGroup>
      <Modal show={showError} onHide={() => setShowError(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Sync error</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <pre className="sync-error"><code>{error}</code></pre>
        </Modal.Body>
      </Modal>
    </div>
  )
}
