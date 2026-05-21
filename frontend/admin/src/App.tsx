import { Routes, Route, Link, useLocation } from 'react-router-dom'
import ProductList from './pages/Product/List'
import ProductCreate from './pages/Product/Create'
import RuleConfig from './pages/Product/RuleConfig'
import AuctionList from './pages/Auction/List'
import AuctionDetail from './pages/Auction/Detail'
import OrderList from './pages/Order/List'

function App() {
  const location = useLocation()

  return (
    <div className="app">
      <nav className="sidebar">
        <h2>竞拍管理后台</h2>
        <ul>
          <li className={location.pathname.startsWith('/products') ? 'active' : ''}>
            <Link to="/products">商品管理</Link>
          </li>
          <li className={location.pathname.startsWith('/auctions') ? 'active' : ''}>
            <Link to="/auctions">竞拍管理</Link>
          </li>
          <li className={location.pathname.startsWith('/orders') ? 'active' : ''}>
            <Link to="/orders">订单管理</Link>
          </li>
        </ul>
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
