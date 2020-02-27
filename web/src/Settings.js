import React from 'react';

import Accounts from './Accounts';
import AdvancedOptions from './AdvancedOptions';
import BalanceSettings from './BalanceSettings';
import Breadcrumb from 'react-bootstrap/Breadcrumb';
import Categories from './Categories';
import { Breadcrumbs, Crumb } from './Breadcrumb';
import { Link, NavLink, Route, Switch } from "react-router-dom";


export default function ({ match }) {
  return (
    <Breadcrumbs as={Breadcrumb} skip={1} render={({ title, match }) =>
      <NavLink className="breadcrumb-item" to={match.url} exact>{title}</NavLink>
    }>
      <Crumb title="Settings" match={match} /> 
      <Switch>
        <Route path={`${match.path}/accounts`} component={Accounts} />
        <Route path={`${match.path}/balances`} component={BalanceSettings} />
        <Route path={`${match.path}/categories`} component={Categories} />
        <Route path={`${match.path}/advanced`} component={AdvancedOptions} />
        <Route exact path={match.path}>
          <Link to={`${match.url}/accounts`}>Accounts</Link>
          <Link to={`${match.url}/balances`}>Balances</Link>
          <Link to={`${match.url}/categories`}>Categories</Link>
          <Link to={`${match.url}/advanced`} className="advanced-settings">Advanced</Link>
        </Route>
      </Switch>
    </Breadcrumbs>
  )
}
