import API from './API';
import React from 'react';
import Button from 'react-bootstrap/Button';
import Spinner from 'react-bootstrap/Spinner';

import './Sync.css';

export default function Sync(props) {
  const [isSyncing, setSyncing] = React.useState(false)
  const [error, setError] = React.useState(null)

  React.useEffect(() => {
    setInterval(() => {
      API.get('/v1/getLedgerSyncStatus')
        .then(res => {
          setSyncing(res.data.Syncing)
          setError(res.data.Error)
        })
    }, 10000)
  }, [])

  const clickSync = () => {
    API.post('/v1/syncLedger')
    setSyncing(true)
  }

  let className = "sync"
  if (props.className) {
    className += " " + props.className
  }
  if (error) {
    className += " sync-failed"
  }

  return (
    <Button
      variant="dark"
      className={className}
      disabled={isSyncing}
      onClick={!isSyncing ? clickSync : null}
    >
      {isSyncing
        ? <Spinner animation="border" size="sm" />
        : (error ? 'Sync Failed' : 'Sync')
      }
    </Button>
  )
}
