import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import reportWebVitals from './reportWebVitals';
import { TestIDProvider } from './context/TestIDContext';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <TestIDProvider>
    {/* <React.StrictMode> */}
      <App />
    {/* </React.StrictMode> */}
  </TestIDProvider>
);

reportWebVitals();
