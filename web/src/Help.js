import React from 'react';
import './Help.css';

import Button from 'react-bootstrap/Button';
import { Crumb } from './Breadcrumb';


const githubFeedback = "https://github.com/JohnStarich/sage/issues/new"
const privateFeedback = "https://forms.gle/iCXiTP9Th5zumbPg9"

export default function Help({ match }) {
  return (
    <div>
      <Crumb title="Help" match={match} />
      <h1>Help &amp; Feedback</h1>
      <p>Send suggestions, request features, or ask questions.</p>
      <p>Choose how to deliver your feedback:</p>
      <div className="feedback-tiles">
        <a href={githubFeedback} target="_blank" rel="noopener noreferrer">
          <Button variant="outline-primary">
            <h2>GitHub</h2>
            <h3><em>Preferred</em></h3>
            <p>Sign in and submit an issue</p>
          </Button>
        </a>
        <a href={privateFeedback} target="_blank" rel="noopener noreferrer">
          <Button variant="outline-secondary">
            <h2>Feedback Form</h2>
            <p>Can include sensitive information</p>
          </Button>
        </a>
      </div>
    </div>
  )
}
