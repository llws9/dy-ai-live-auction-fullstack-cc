import { Routes, Route, Link, useLocation, Navigate } from 'react-router-dom'
import { AdminAuthProvider, useAdminAuth } from './store/authContext'
import AdminLogin from './pages/Login'
import ProductList from './pages/Product/List'
import ProductCreate from './pages/Product/Create'
import RuleConfig from './pages/Product/RuleConfig'
import AuctionList from './pages/Auction/List'
import AuctionDetail from './pages/Auction/Detail'
import OrderList from './pages/Order/List'

// 管理员认证保护组件
function PrivateRoute({ children }: { children: React.ReactElement }) {
  const { isAuthenticated, isAdmin, loading } = useAdminAuth()

  if (loading) {
    return <div style={{ padding: '20px', textAlign: 'center' }}>加载中...</div>
  }

  if (!isAuthenticated || !isAdmin) {
    return <Navigate to="/admin-login" replace />
  }

  return children
}

function AppContent() {
  const location = useLocation()

  const navItems = [
    { path: '/products', label: '商品管理', icon: '📦' },
    { path: '/auctions', label: '竞拍管理', icon: '🎯' },
    { path: '/orders', label: '订单管理', icon: '🧾' },
  ]

  return (
    <div className="app">
      <nav className="sidebar">
        <div className="sidebar-header">
          <div className="sidebar-logo">
            <div className="sidebar-logo-icon">🎯</div>
            <h2>竞拍管理后台</h2>
          </div>
        </div>
        <div className="sidebar-nav">
          <ul>
            {navItems.map((item) => (
              <li
                key={item.path}
                className={location.pathname.startsWith(item.path) ? 'active' : ''}
              >
                <Link to={item.path}>
                  <span>{item.icon}</span>
                  <span>{item.label}</span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </nav>
      <main className="content">
        <Routes>
          <Route path="/" element={
            <PrivateRoute>
              <ProductList />
            </PrivateRoute>
          } />
          <Route path="/products" element={
            <PrivateRoute>
              <ProductList />
            </PrivateRoute>
          } />
          <Route path="/products/create" element={
            <PrivateRoute>
              <ProductCreate />
            </PrivateRoute>
          } />
          <Route path="/products/:id/rules" element={
            <PrivateRoute>
              <RuleConfig />
            </PrivateRoute>
          } />
          <Route path="/auctions" element={
            <PrivateRoute>
              <AuctionList />
            </PrivateRoute>
          } />
          <Route path="/auctions/:id" element={
            <PrivateRoute>
              <AuctionDetail />
            </PrivateRoute>
          } />
          <Route path="/orders" element={
            <PrivateRoute>
              <OrderList />
            </PrivateRoute>
          } />
        </Routes>
      </main>
    </div>
  )
}

function App() {
  return (
    <AdminAuthProvider>
      <Routes>
        <Route path="/admin-login" element={<AdminLogin />} />
        <Route path="/*" element={<AppContent />} />
      </Routes>
    </AdminAuthProvider>
  )
}

export default App
