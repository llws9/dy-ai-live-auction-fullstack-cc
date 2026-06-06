import { Routes, Route, Navigate } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import { AuthProvider, useAuth } from './store/authContext'
import { AuctionProvider } from './store/auctionContext'
import { GrowthBookContextProvider } from './store/growthbookContext'
import { ThemeProvider } from './store/themeContext'
import ErrorBoundary from './components/ErrorBoundary'
import { ToastProvider, useToast } from './components/Toast'
import { setToastFunction } from './services/api'
import { errorMonitor } from './utils/errorMonitor'
import { useEffect } from 'react'
import LoadingSpinner from './components/LoadingSpinner'
import MobileContainer from './components/MobileShell/MobileContainer'
import { LegacyAuctionRedirect, LegacyResultRedirect } from './routes/legacyRedirects'

// 动态导入路由组件
const Login = lazy(() => import('./pages/Login'))
const Home = lazy(() => import('./pages/Home'))
const ProductDetail = lazy(() => import('./pages/ProductDetail'))
const Result = lazy(() => import('./pages/Result'))
const History = lazy(() => import('./pages/History'))
const Live = lazy(() => import('./pages/Live'))
const Follow = lazy(() => import('./pages/Follow'))
const Profile = lazy(() => import('./pages/User/Index'))
const Notifications = lazy(() => import('./pages/Notifications'))
const Addresses = lazy(() => import('./pages/Addresses'))
const OrderList = lazy(() => import('./pages/Order/List'))
const OrderDetail = lazy(() => import('./pages/Order/Detail'))

// 认证保护组件
function PrivateRoute({ children }: { children: React.ReactElement }) {
  const { isAuthenticated, loading } = useAuth()

  if (loading) {
    return <LoadingSpinner />
  }

  return isAuthenticated ? children : <Navigate to="/login" replace />
}

// 初始化 Toast 函数的组件
function ToastInitializer() {
  const { showToast } = useToast()

  useEffect(() => {
    setToastFunction(showToast)
  }, [showToast])

  return null
}

// 初始化错误监控的组件
function ErrorMonitorInitializer() {
  const { user } = useAuth()

  useEffect(() => {
    // 当用户登录时，设置用户信息到错误监控
    if (user) {
      errorMonitor.setUser(user.id, user.role)
    } else {
      errorMonitor.clearUser()
    }
  }, [user])

  return null
}

function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <ToastProvider>
          <ToastInitializer />
          <AuthProvider>
            <ErrorMonitorInitializer />
            <GrowthBookContextProvider>
              <AuctionProvider>
                <MobileContainer>
                  <Suspense fallback={<LoadingSpinner />}>
                    <Routes>
                      <Route path="/login" element={<Login />} />
                      <Route path="/" element={<Home />} />
                      <Route path="/live" element={<Live />} />
                      <Route path="/detail" element={<ProductDetail />} />
                      <Route path="/auction/:id" element={<LegacyAuctionRedirect />} />
                      <Route path="/result" element={<Result />} />
                      <Route path="/result/:id" element={<LegacyResultRedirect />} />
                      <Route path="/profile" element={
                        <PrivateRoute>
                          <Profile />
                        </PrivateRoute>
                      } />
                      <Route path="/notifications" element={
                        <PrivateRoute>
                          <Notifications />
                        </PrivateRoute>
                      } />
                      <Route path="/following" element={
                        <PrivateRoute>
                          <Follow />
                        </PrivateRoute>
                      } />
                      <Route path="/follow" element={<Navigate to="/following" replace />} />
                      <Route path="/history" element={
                        <PrivateRoute>
                          <History />
                        </PrivateRoute>
                      } />
                      <Route path="/addresses" element={
                        <PrivateRoute>
                          <Addresses />
                        </PrivateRoute>
                      } />
                      <Route path="/orders" element={
                        <PrivateRoute>
                          <OrderList />
                        </PrivateRoute>
                      } />
                      <Route path="/order/:id" element={
                        <PrivateRoute>
                          <OrderDetail />
                        </PrivateRoute>
                      } />
                    </Routes>
                  </Suspense>
                </MobileContainer>
              </AuctionProvider>
            </GrowthBookContextProvider>
          </AuthProvider>
        </ToastProvider>
      </ThemeProvider>
    </ErrorBoundary>
  )
}

export default App
