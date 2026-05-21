import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './store/authContext'
import Login from './pages/Login'
import Home from './pages/Home'
import Auction from './pages/Auction'
import Result from './pages/Result'
import History from './pages/History'
import Live from './pages/Live'
import { AuctionProvider } from './store/auctionContext'

// 认证保护组件
function PrivateRoute({ children }: { children: React.ReactElement }) {
  const { isAuthenticated, loading } = useAuth()

  if (loading) {
    return <div style={{ padding: '20px', textAlign: 'center' }}>加载中...</div>
  }

  return isAuthenticated ? children : <Navigate to="/login" replace />
}

function App() {
  return (
    <AuthProvider>
      <AuctionProvider>
        <div className="app">
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<Home />} />
            <Route path="/live" element={<Live />} />
            <Route path="/auction/:id" element={<Auction />} />
            <Route path="/result/:id" element={<Result />} />
            <Route path="/history" element={
              <PrivateRoute>
                <History />
              </PrivateRoute>
            } />
          </Routes>
        </div>
      </AuctionProvider>
    </AuthProvider>
  )
}

export default App
