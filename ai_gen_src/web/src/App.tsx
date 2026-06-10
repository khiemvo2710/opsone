import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Dashboard } from './pages/Dashboard';
import { Settings } from './pages/Settings';
import { AgentChanges } from './pages/AgentChanges';
import { MaintenancePage } from './pages/MaintenancePage';
import { IncidentDetail } from './pages/IncidentDetail';
import { IncidentsPage } from './pages/IncidentsPage';

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="incidents" element={<IncidentsPage />} />
          <Route path="settings" element={<Settings />} />
          <Route path="changes" element={<AgentChanges />} />
          <Route path="maintenance" element={<MaintenancePage />} />
          <Route path="incidents/:id" element={<IncidentDetail />} />
        </Route>
        <Route path="auth/callback" element={<Navigate to="/" replace />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
