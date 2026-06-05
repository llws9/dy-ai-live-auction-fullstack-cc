import React from "react"
import { Link, useLocation, useNavigate } from "react-router-dom"
import {
  LayoutDashboard,
  Package,
  Gavel,
  Video,
  ShoppingBag,
  BarChart3,
  Settings,
  LogOut,
  ChevronRight,
  Menu,
  Bell,
  Search
} from "lucide-react"
import { cn } from "@/lib/utils"
import { Avatar } from "./ui/avatar"
import { useAuth } from "@/shared/auth"
import { ADMIN_ROLE, MERCHANT_ROLE, isAllowedRole, roleLabel } from "@/shared/auth/roles"

interface NavItem {
  title: string
  path: string
  icon: React.ElementType
  allowedRoles?: number[]
  children?: {
    title: string
    path: string
    allowedRoles?: number[]
    titleByRole?: Partial<Record<number, string>>
  }[]
}

const navItems: NavItem[] = [
  { title: "经营总览", path: "/dashboard", icon: LayoutDashboard },
  {
    title: "商品管理",
    path: "/goods",
    icon: Package,
    allowedRoles: [MERCHANT_ROLE],
    children: [
      { title: "商品列表", path: "/goods/list" },
      { title: "创建商品", path: "/goods/create", allowedRoles: [MERCHANT_ROLE] },
    ]
  },
  {
    title: "竞拍管理",
    path: "/auction",
    icon: Gavel,
    children: [
      { title: "竞拍列表", path: "/auction/list" },
      { title: "规则模板", path: "/auction/rules", allowedRoles: [MERCHANT_ROLE] },
    ]
  },
  {
    title: "直播间管理",
    path: "/live",
    icon: Video,
    children: [
      { title: "直播间列表", path: "/live/list", titleByRole: { [MERCHANT_ROLE]: "我的直播间" } },
      { title: "一口价上下架", path: "/live/fixed-price", allowedRoles: [MERCHANT_ROLE] },
      { title: "创建直播间", path: "/live/create", allowedRoles: [MERCHANT_ROLE] },
    ]
  },
  {
    title: "订单管理",
    path: "/order",
    icon: ShoppingBag,
    children: [
      { title: "订单列表", path: "/order/list" },
    ]
  },
  {
    title: "数据统计",
    path: "/stats",
    icon: BarChart3,
    children: [
      { title: "竞拍统计", path: "/stats/auction" },
      { title: "收入统计", path: "/stats/revenue" },
      { title: "用户统计", path: "/stats/user", allowedRoles: [ADMIN_ROLE] },
    ]
  },
  {
    title: "系统设置",
    path: "/system",
    icon: Settings,
    children: [
      { title: "个人中心", path: "/system/profile" },
      { title: "角色管理", path: "/system/permission/roles", allowedRoles: [ADMIN_ROLE] },
      { title: "用户管理", path: "/system/permission/users", allowedRoles: [ADMIN_ROLE] },
    ]
  },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [expandedMenus, setExpandedMenus] = React.useState<string[]>([])
  const visibleNavItems = React.useMemo(() => navItems
    .map((item) => {
      const visibleChildren = item.children
        ?.filter((child) => isAllowedRole(child.allowedRoles, user?.role))
        .map((child) => ({
          ...child,
          title: child.titleByRole?.[user?.role ?? 0] ?? child.title,
        }))
      if (item.children) {
        return visibleChildren && visibleChildren.length > 0 && isAllowedRole(item.allowedRoles, user?.role)
          ? { ...item, children: visibleChildren }
          : null
      }
      return isAllowedRole(item.allowedRoles, user?.role) ? item : null
    })
    .filter((item): item is NavItem => item !== null), [user?.role])

  const toggleMenu = (title: string) => {
    setExpandedMenus(prev =>
      prev.includes(title) ? prev.filter(t => t !== title) : [...prev, title]
    )
  }

  // Auto expand menu based on current path
  React.useEffect(() => {
    visibleNavItems.forEach(item => {
      if (item.children?.some(child => location.pathname === child.path)) {
        if (!expandedMenus.includes(item.title)) {
          setExpandedMenus(prev => [...prev, item.title])
        }
      }
    })
  }, [location.pathname, visibleNavItems])

  const handleLogout = () => {
    localStorage.removeItem('admin_auth_token');
    localStorage.removeItem('admin_auth_user');
    navigate("/admin-login")
  }

  return (
    <div className="flex h-screen w-full overflow-hidden bg-[#f8fafc]">
      {/* Sidebar */}
      <aside className="w-64 bg-[#0f172a] text-slate-300 flex flex-col shrink-0 border-r border-slate-800 shadow-2xl z-20">
        <div className="h-16 flex items-center px-6 border-b border-slate-800 gap-3">
          <div className="w-8 h-8 rounded-lg bg-amber-500 flex items-center justify-center">
            <Gavel className="w-5 h-5 text-[#0f172a]" />
          </div>
          <span className="font-bold text-lg text-white tracking-tight">直播竞拍后台</span>
        </div>

        <nav className="flex-1 overflow-y-auto py-6 no-scrollbar">
          <div className="px-4 space-y-1">
            {visibleNavItems.map((item) => (
              <div key={item.title} className="space-y-1">
                {item.children ? (
                  <>
                    <button
                      onClick={() => toggleMenu(item.title)}
                      className={cn(
                        "w-full flex items-center justify-between px-3 py-2.5 rounded-lg transition-all duration-200 hover:bg-slate-800 hover:text-white group",
                        expandedMenus.includes(item.title) ? "text-white bg-slate-800/50" : ""
                      )}
                    >
                      <div className="flex items-center gap-3">
                        <item.icon className={cn("w-5 h-5", expandedMenus.includes(item.title) ? "text-amber-500" : "text-slate-400 group-hover:text-amber-400")} />
                        <span className="text-sm font-medium">{item.title}</span>
                      </div>
                      <ChevronRight className={cn("w-4 h-4 transition-transform", expandedMenus.includes(item.title) ? "rotate-90" : "")} />
                    </button>
                    {expandedMenus.includes(item.title) && (
                      <div className="mt-1 ml-4 pl-4 border-l border-slate-800 space-y-1">
                        {item.children.map((child) => (
                          <Link
                            key={child.path}
                            to={child.path}
                            className={cn(
                              "block px-3 py-2 rounded-md text-sm transition-all",
                              location.pathname === child.path
                                ? "text-amber-500 bg-amber-500/10 font-medium"
                                : "text-slate-400 hover:text-white hover:bg-slate-800"
                            )}
                          >
                            {child.title}
                          </Link>
                        ))}
                      </div>
                    )}
                  </>
                ) : (
                  <Link
                    to={item.path}
                    className={cn(
                      "flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 group",
                      location.pathname === item.path
                        ? "bg-amber-500 text-[#0f172a] font-semibold shadow-lg shadow-amber-500/20"
                        : "hover:bg-slate-800 hover:text-white"
                    )}
                  >
                    <item.icon className={cn("w-5 h-5", location.pathname === item.path ? "text-[#0f172a]" : "text-slate-400 group-hover:text-amber-400")} />
                    <span className="text-sm font-medium">{item.title}</span>
                  </Link>
                )}
              </div>
            ))}
          </div>
        </nav>

        <div className="p-4 border-t border-slate-800">
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-all"
          >
            <LogOut className="w-5 h-5" />
            <span className="text-sm font-medium">退出登录</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Header */}
        <header className="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8 shrink-0 z-10">
          <div className="flex items-center gap-4">
            <button className="lg:hidden text-slate-500">
              <Menu className="w-6 h-6" />
            </button>
            <div className="relative hidden md:block">
              <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="搜索商品、订单、竞拍..."
                className="pl-10 pr-4 py-2 bg-slate-100 border-none rounded-full text-sm w-64 focus:ring-2 focus:ring-amber-500 outline-none transition-all"
              />
            </div>
          </div>

          <div className="flex items-center gap-6">
            <button className="relative text-slate-500 hover:text-amber-600 transition-colors">
              <Bell className="w-5 h-5" />
              <span className="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full border-2 border-white"></span>
            </button>
            <div className="flex items-center gap-3 pl-6 border-l border-slate-200">
              <div className="text-right hidden sm:block">
                <p className="text-sm font-semibold text-slate-900 leading-tight">{user?.name || '未登录用户'}</p>
                <p className="text-xs text-slate-500">{roleLabel(user?.role)}</p>
              </div>
              <Avatar className="w-10 h-10 rounded-full border-2 border-amber-500/20" />
            </div>
          </div>
        </header>

        {/* Page Area */}
        <main className="flex-1 overflow-y-auto p-8 no-scrollbar">
          <div className="max-w-7xl mx-auto space-y-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}
