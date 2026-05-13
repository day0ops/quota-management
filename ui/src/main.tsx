import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { Global } from '@emotion/react';
import { Toaster } from 'react-hot-toast';
import { globalStyles } from './styles';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider } from './contexts/ThemeContext';
import App from './App';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Global styles={globalStyles} />
    <Toaster
      position="top-right"
      toastOptions={{
        duration: 4000,
        style: {
          background: 'var(--color-card-bg)',
          color: 'var(--color-foreground)',
          border: '1px solid var(--color-border)',
        },
        success: {
          iconTheme: {
            primary: '#22C55E',
            secondary: 'var(--color-foreground)',
          },
        },
        error: {
          iconTheme: {
            primary: '#EF4444',
            secondary: 'var(--color-foreground)',
          },
        },
      }}
    />
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <App />
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  </StrictMode>
);
