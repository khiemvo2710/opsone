import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { ErrorBoundary } from './components/ErrorBoundary';
import { Layout } from './components/Layout';
import { Dashboard } from './pages/Dashboard';
import { Settings } from './pages/Settings';
import { MaintenancePage } from './pages/MaintenancePage';
import { IncidentsPage } from './pages/IncidentsPage';

export function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="incidents" element={<IncidentsPage />} />
            <Route path="settings" element={<Settings />} />
            <Route path="maintenance" element={<MaintenancePage />} />
          </Route>
          <Route path="auth/callback" element={<Navigate to="/" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
