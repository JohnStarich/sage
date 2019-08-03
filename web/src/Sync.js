import axios from 'axios';
import React from 'react';
import Button from 'react-bootstrap/Button';
import Spinner from 'react-bootstrap/Spinner';

import './Sync.css';

function runSync() {
  return axios.post('/api/v1/sync')
    .then(res => res.data)
}

export default function Sync(props) {
  const [isSyncing, setSyncing] = React.useState(false)
  const [failed, setFailed] = React.useState(false)

  const clickSync = () => {
    setSyncing(true)
    runSync().then(() => {
      if (props.onSync) {
        props.onSync()
      }
      setFailed(false)
    }).catch(e => {
      console.error(e)
      setFailed(true)
    }).finally(() => {
      setSyncing(false)
    })
  }

  let className = "sync"
  if (props.className) {
    className += " " + props.className
  }
  if (failed) {
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
        : (failed ? 'Sync Failed' : 'Sync')
      }
    </Button>
  )
}
