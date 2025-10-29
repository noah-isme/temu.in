
/// <reference types="vite/client" />

import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import App from './App';
import './styles/index.css';

async function prepare() {
  if (import.meta.env.DEV) {
    // Start MSW worker in development so frontend can work without backend
    const { worker } = await import('./mocks/browser');
    await worker.start({ onUnhandledRequest: 'bypass' });
    // Helpful log to confirm MSW is running (check browser devtools console)
    // eslint-disable-next-line no-console
    console.info('[MSW] Mock Service Worker started (development mode)');
  }
}

const queryClient = new QueryClient();

prepare().then(() => {
  ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <App />
        <ToastContainer position="top-right" autoClose={3000} toastClassName="toastify-toast" progressClassName="Toastify__progress-bar" />
      </QueryClientProvider>
    </React.StrictMode>
  );
});
