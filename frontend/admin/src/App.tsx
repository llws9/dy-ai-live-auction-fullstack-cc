import { Routes, Route, useLocation, Navigate } from "react-router-dom"
import { Layout } from "@/components/Layout"
import Login from "@/pages-new/Login"
import Dashboard from "@/pages-new/Dashboard"
import GoodsList from "@/pages-new/GoodsList"
import GoodsEdit from "@/pages-new/GoodsEdit"
import AuctionList from "@/pages-new/AuctionList"
import AuctionDetail from "@/pages-new/AuctionDetail"
import AuctionRules from "@/pages-new/AuctionRules"
import LiveList from "@/pages-new/LiveList"
import LiveDetail from "@/pages-new/LiveDetail"
import OrderList from "@/pages-new/OrderList"
import OrderDetail from "@/pages-new/OrderDetail"
import Stats from "@/pages-new/Stats"
import Profile from "@/pages-new/Profile"
import Permissions from "@/pages-new/Permissions"
import { RequireAuth, AuthProvider } from "@/shared/auth"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { GrowthBookContextProvider } from "@/shared/growthbook"

function AppContent() {
  const location = useLocation()
  const isLoginPage = location.pathname === "/admin-login" || location.pathname === "/"

  if (isLoginPage) {
    return (
      <Routes>
        <Route path="/admin-login" element={<Login />} />
        <Route path="/" element={<Navigate to="/admin-login" replace />} />
      </Routes>
    )
  }

  return (
    <RequireAuth>
      <Layout>
        <Routes>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/goods/list" element={<GoodsList />} />
          <Route path="/goods/create" element={<GoodsEdit />} />
          <Route path="/goods/edit" element={<GoodsEdit />} />
          <Route path="/auction/list" element={<AuctionList />} />
          <Route path="/auction/detail" element={<AuctionDetail />} />
          <Route path="/auction/rules" element={<AuctionRules />} />
          <Route path="/auction/rules/create" element={<AuctionRules />} />
          <Route path="/auction/rules/edit" element={<AuctionRules />} />
          <Route path="/live/list" element={<LiveList />} />
          <Route path="/live/detail" element={<LiveDetail />} />
          <Route path="/live/create" element={<LiveList />} />
          <Route path="/order/list" element={<OrderList />} />
          <Route path="/order/detail" element={<OrderDetail />} />
          <Route path="/stats/auction" element={<Stats />} />
          <Route path="/stats/revenue" element={<Stats />} />
          <Route path="/stats/user" element={<Stats />} />
          <Route path="/system/profile" element={<Profile />} />
          <Route path="/system/permission/roles" element={<Permissions />} />
          <Route path="/system/permission/users" element={<Permissions />} />
          {/* Legacy routes - redirect to new paths */}
          <Route path="/products" element={<Navigate to="/goods/list" replace />} />
          <Route path="/auctions" element={<Navigate to="/auction/list" replace />} />
          <Route path="/live-streams" element={<Navigate to="/live/list" replace />} />
          <Route path="/orders" element={<Navigate to="/order/list" replace />} />
          <Route path="/statistics" element={<Navigate to="/stats/auction" replace />} />
        </Routes>
      </Layout>
    </RequireAuth>
  )
}

export default function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <GrowthBookContextProvider>
          <AppContent />
        </GrowthBookContextProvider>
      </AuthProvider>
    </ErrorBoundary>
  )
}
