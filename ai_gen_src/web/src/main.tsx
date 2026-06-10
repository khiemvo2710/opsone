import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MsalProvider } from '@azure/msal-react';
import { App } from './App';
import { ToastProvider } from './context/ToastContext';
import { devAuthBypass, msalEnabled, pca } from './auth/msalConfig';
import './index.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 10_000 },
  },
});

async function bootstrap() {
  if (msalEnabled) {
    await pca.initialize();
  }

  const tree = (
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          {msalEnabled ? (
            <MsalProvider instance={pca}>
              <App />
            </MsalProvider>
          ) : (
            <App />
          )}
        </ToastProvider>
      </QueryClientProvider>
    </StrictMode>
  );

  createRoot(document.getElementById('root')!).render(tree);

  if (devAuthBypass) {
    console.info('OpsOne UI: DEV_AUTH_BYPASS — dùng X-OpsOne-Role header');
  }
}

void bootstrap();
