import React from 'react';
import { useNavigate } from 'react-router-dom';
import { logout } from '../services/auth';

function Dashboard() {
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  return (
    <div>
      <h1>Dashboard</h1>
      <p>Welcome to your dashboard!</p>
      <button onClick={handleLogout}>Log Out</button>
    </div>
  );
}

export default Dashboard;
