import { NavLink, Outlet } from 'react-router-dom';
import { HealthBadge } from './HealthBadge';
import { ChatWidget } from './ChatWidget';
import { useSSE } from '../hooks/useSSE';
import { useOverallHealth } from '../hooks/useOverallHealth';

const NAV = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/incidents', label: 'Sự cố' },
  { to: '/maintenance', label: 'Bảo trì' },
  { to: '/changes', label: 'Lịch sử' },
  { to: '/settings', label: 'Cấu hình' },
];

export function Layout() {
  useSSE();
  const overallHealth = useOverallHealth();

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-header__brand">
          <strong>OpsOne</strong>
          <HealthBadge
            status={overallHealth.status}
            label={overallHealth.label}
            size="sm"
          />
        </div>

        <nav className="top-nav" aria-label="Menu chinh">
          {NAV.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `top-nav__link${isActive ? ' top-nav__link--active' : ''}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </header>

      <main className="main-content">
        <Outlet />
      </main>

      <ChatWidget />
    </div>
  );
}
