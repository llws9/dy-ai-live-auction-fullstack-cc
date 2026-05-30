import { NavLink, Outlet } from 'react-router-dom';

const navItemStyle: React.CSSProperties = {
  display: 'block',
  padding: '10px 16px',
  borderRadius: 'var(--radius-md, 6px)',
  color: 'var(--color-text, #1f2937)',
  textDecoration: 'none',
  marginBottom: 4,
};

const activeStyle: React.CSSProperties = {
  background: 'var(--color-primary, #3b82f6)',
  color: '#fff',
};

export default function Layout() {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <aside
        style={{
          width: 220,
          padding: 16,
          background: 'var(--color-bg-soft, #f8fafc)',
          borderRight: '1px solid #e5e7eb',
        }}
      >
        <h2 style={{ fontSize: 16, marginBottom: 16 }}>测试演示平台</h2>
        <nav>
          <NavLink
            to="/test"
            end
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            控制台
          </NavLink>
          <NavLink
            to="/test/pressure"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            压力测试
          </NavLink>
          <NavLink
            to="/test/e2e"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            E2E 全链路
          </NavLink>
          <NavLink
            to="/test/antisnipe"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            防狙击 (F)
          </NavLink>
          <NavLink
            to="/test/callback"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            回调投递 (H)
          </NavLink>
          <NavLink
            to="/test/chaos"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            故障注入 (G)
          </NavLink>
          <NavLink
            to="/test/compare"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            A/B 对比
          </NavLink>
          <NavLink
            to="/test/screen"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            演示大屏
          </NavLink>
          <NavLink
            to="/test/history"
            style={({ isActive }) => ({ ...navItemStyle, ...(isActive ? activeStyle : {}) })}
          >
            历史记录
          </NavLink>
        </nav>
      </aside>
      <main style={{ flex: 1, padding: 24, overflow: 'auto' }}>
        <Outlet />
      </main>
    </div>
  );
}
