import React from 'react';
import API from './API';
import { BrowserRouter as Router, Route, Switch, NavLink } from "react-router-dom";
import 'bootstrap/dist/css/bootstrap.min.css';
import './common.css';
import './App.css';

import Activity from './Activity';
import Budgets from './Budgets';
import Login from './Login';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import Settings from './Settings';
import Sync from './Sync';
import { Crumb } from './Breadcrumb';
import { ReactComponent as Logo } from './logo/sage.svg';

function App() {
  return (
    <Router basename="/web">
      <Route path="/" component={AppContent} />
    </Router>
  );
}

export default App;

function AppContent({ match }) {
  const [version, setVersion] = React.useState(null)
  const [needsUpdate, setNeedsUpdate] = React.useState(false)
  const [syncTime, setSyncTime] = React.useState(new Date())
  const [appClasses, setAppClasses] = React.useState([])

  React.useEffect(() => {
    API.get('/v1/getVersion')
      .then(res => {
        let version = res.data.Version
        if (version === "") {
          version = "dev"
        }
        setVersion(version)
        setNeedsUpdate(res.data.UpdateAvailable)
      })
    if (navigator.userAgent.includes('Sage/1.0.0') && navigator.userAgent.includes('Macintosh')) {
      setAppClasses(["electron"])
    }
  }, [])

  return (
    <div className={["app"].concat(appClasses).join(" ")}>
      <Crumb title="Sage" match={match} />
      <Navbar className="sage-nav" bg="dark" expand="sm" variant="dark" sticky="top">
        <NavLink exact to="/">
          <Navbar.Brand>
            <Logo className="sage-logo dark" />
            <div className="sage-title">
              <span className="sage-name">Sage</span>
              <span className="sage-version-container" onClick={e => {
                    // Creates a detached element and clicks it. This is required to
                    // 1) avoid nested link issues with NavLink
                    // 2) and correctly open a new window in Electron
                    e.preventDefault()
                    let elem = document.createElement('a')
                    elem.setAttribute('href', 'https://github.com/JohnStarich/sage/blob/master/.github/docs/download-app.md')
                    elem.setAttribute('target', '_blank')
                    elem.click()
                  }}>
                <span className="sage-version">{version}</span>
                {needsUpdate ?
                  <span className="needs-update"></span>
                : null}
              </span>
            </div>
          </Navbar.Brand>
        </NavLink>
        <Navbar.Toggle />
        <Navbar.Collapse>
          <Nav className="mr-auto">
            <NavLink exact to="/" className="nav-link">Activity</NavLink>
            <NavLink exact to="/budgets" className="nav-link">Budgets</NavLink>
          </Nav>
          <Nav>
            <Sync className="mr-2" onSync={() => setSyncTime(new Date())} />
            <NavLink to="/settings" aria-label="Settings" className="nav-link settings-icon">⚙</NavLink>
          </Nav>
        </Navbar.Collapse>
      </Navbar>

      <div className="content">
        <Switch>
          <Route path="/" exact component={() => <Activity syncTime={syncTime} />} />
          <Route path="/login" component={Login} />
          <Route path="/budgets" component={Budgets} />
          <Route path="/settings" component={Settings} />
        </Switch>
      </div>
    </div>
  )
}
