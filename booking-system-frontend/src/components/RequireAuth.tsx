import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { getAuthToken } from '../api';

export default function RequireAuth({ children }: { children: JSX.Element }) {
  const token = getAuthToken();
  const location = useLocation();

  if (!token) {
    // Redirect to login preserving the location user was trying to reach
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return children;
}
