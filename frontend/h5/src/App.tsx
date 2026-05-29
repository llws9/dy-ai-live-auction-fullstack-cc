import { Routes, Route, Navigate } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import { AuthProvider, useAuth } from './store/authContext'
import { AuctionProvider } from './store/auctionContext'
import { GrowthBookContextProvider } from './store/growthbookContext'
import ErrorBoundary from './components/ErrorBoundary'
import { ToastProvider, useToast } from './components/Toast'
import { setToastFunction } from './services/api'
import { errorMonitor } from './utils/errorMonitor'
import { useEffect, useState } from 'react'
import LoadingSpinner from './components/LoadingSpinner'
import LiveReminderModal, { StreamInfo } from './components/LiveReminderModal'

// 动态导入路由组件
const Login = lazy(() => import('./pages/Login'))
const Home = lazy(() => import('./pages/Home'))
const Auction = lazy(() => import('./pages/Auction'))
const Result = lazy(() => import('./pages/Result'))
const History = lazy(() => import('./pages/History'))
const Live = lazy(() => import('./pages/Live'))
const Follow = lazy(() => import('./pages/Follow'))

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
  const [isReminderOpen, setIsReminderOpen] = useState(false)
  const [activeStream, setActiveStream] = useState<StreamInfo | null>(null)

  useEffect(() => {
    // 模拟应用启动时检测是否有关注的直播间正在开播
    const checkLiveStatus = () => {
      // 模拟后端返回的数据
      const mockStream: StreamInfo = {
        id: 1,
        name: "高奢珠宝首发专场",
        avatarUrl: "https://images.unsplash.com/photo-1548036328-c9fa89d128fa?w=150",
        statusText: "正在直播"
      }
      setActiveStream(mockStream)
      setIsReminderOpen(true)
    }

    // 延迟 1.5 秒后弹出，模拟网络请求和应用初始化
    const timer = setTimeout(checkLiveStatus, 1500)
    return () => clearTimeout(timer)
  }, [])

  return (
    <ErrorBoundary>
      <ToastProvider>
        <ToastInitializer />
        <AuthProvider>
          <ErrorMonitorInitializer />
          <GrowthBookContextProvider>
            <AuctionProvider>
              <div className="app">
                <Suspense fallback={<LoadingSpinner />}>
                  <Routes>
                    <Route path="/login" element={<Login />} />
                    <Route path="/" element={<Home />} />
                    <Route path="/live" element={<Live />} />
                    <Route path="/auction/:id" element={<Auction />} />
                    <Route path="/result/:id" element={<Result />} />
                    <Route path="/follow" element={
                      <PrivateRoute>
                        <Follow />
                      </PrivateRoute>
                    } />
                    <Route path="/history" element={
                      <PrivateRoute>
                        <History />
                      </PrivateRoute>
                    } />
                  </Routes>
                </Suspense>

                {/* 全局挂载开播提醒弹窗 */}
                <LiveReminderModal 
                  isOpen={isReminderOpen} 
                  onClose={() => setIsReminderOpen(false)} 
                  stream={activeStream} 
                />
              </div>
            </AuctionProvider>
          </GrowthBookContextProvider>
        </AuthProvider>
      </ToastProvider>
    </ErrorBoundary>
  )
}

export default App
