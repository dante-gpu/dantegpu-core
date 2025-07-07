import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './styles/globals.css';

// Import Buffer from the polyfilled module to ensure it's available
import { Buffer } from 'buffer';

// Ensure Buffer is available globally for any modules that need it
if (typeof (globalThis as any).Buffer === 'undefined') {
  (globalThis as any).Buffer = Buffer;
  (window as any).Buffer = Buffer;
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
); 