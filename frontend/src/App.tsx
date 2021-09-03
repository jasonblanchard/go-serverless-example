import React from 'react';
import {
  BrowserRouter as Router,
  Switch,
  Route
} from "react-router-dom";
import {
  RecoilRoot,
} from 'recoil';

import HomePage from './HomePage';
import LoginPage from './LoginPage';

function App() {
  return (
    <RecoilRoot>
      <Router>
        <Switch>
          <Route path="/login" component={LoginPage} />
          <Route path="/home" component={HomePage} />
        </Switch>
      </Router>
    </RecoilRoot>
  );
}

export default App;
