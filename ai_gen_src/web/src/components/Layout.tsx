import { NavLink, Outlet } from 'react-router-dom';
import { ASSISTANT_NAME } from '../utils/assistantIdentity';
import { HealthBadge } from './HealthBadge';
import { ChatWidget } from './ChatWidget';
import { ChatAvatar } from './ChatAvatar';
import { useSSE } from '../hooks/useSSE';
import { useOverallHealth } from '../hooks/useOverallHealth';
import { useAuth } from '../context/AuthContext';
import { inferProfileFromSingleMessage } from '../utils/chatUserProfile';

const NAV = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/incidents', label: 'Sự cố' },
  { to: '/maintenance', label: 'Bảo trì' },
  { to: '/settings', label: 'Cấu hình' },
];

export function Layout() {
  useSSE();
  const overallHealth = useOverallHealth();
  const { session, logout } = useAuth();

  const userProfile = session?.name
    ? (() => {
        const p = inferProfileFromSingleMessage(session.name);
        if (!p.displayName) p.displayName = session.name;
        return p;
      })()
    : {};

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-header__brand">
          <img
            src="/favicon-64.png"
            alt={ASSISTANT_NAME}
            className="app-header__logo"
            width={32}
            height={32}
          />
          <strong>{ASSISTANT_NAME}</strong>
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

        {session && (
          <div className="app-header__user">
            <div className="app-header__user-avatar">
              <ChatAvatar role="user" userProfile={userProfile} sessionSeed={session.name} />
            </div>
            <span className="app-header__user-name">{session.name}</span>
            <button
              type="button"
              className="app-header__logout-btn"
              onClick={logout}
              title="Đăng xuất"
            >
              Đăng xuất
            </button>
          </div>
        )}
      </header>

      <main className="main-content">
        <Outlet />
      </main>

      <ChatWidget />
    </div>
  );
}
