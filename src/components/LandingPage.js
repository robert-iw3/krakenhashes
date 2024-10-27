import React from 'react';
import { Link } from 'react-router-dom';

function LandingPage() {
  return (
    <div>
      <h1>Welcome to HashDom</h1>
      <p>Please log in to access your dashboard.</p>
      <Link to="/login">Log In</Link>
    </div>
  );
}

export default LandingPage;
