import { Routes, Route, Link, useLocation } from 'react-router-dom'
import ProductList from './pages/Product/List'
import ProductCreate from './pages/Product/Create'
import RuleConfig from './pages/Product/RuleConfig'
import AuctionList from './pages/Auction/List'
import AuctionDetail from './pages/Auction/Detail'
import OrderList from './pages/Order/List'

function App() {
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
          <Route path="/" element={<ProductList />} />
          <Route path="/products" element={<ProductList />} />
          <Route path="/products/create" element={<ProductCreate />} />
          <Route path="/products/:id/rules" element={<RuleConfig />} />
          <Route path="/auctions" element={<AuctionList />} />
          <Route path="/auctions/:id" element={<AuctionDetail />} />
          <Route path="/orders" element={<OrderList />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
