import { User, Mail, Phone, Shield, ShieldCheck, Key, Bell } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { useAuth } from "@/shared/auth"

export default function Profile() {
  const { user } = useAuth()
  // 用户信息（从AuthContext获取）
  const userInfo = user || {
    name: '管理员',
    email: '',
    phone: '',
    avatar: '',
    role: 2,
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">个人中心</h1>
          <p className="text-sm text-slate-500">管理您的账号信息与安全设置</p>
        </div>
        {/* 保存更改 - 后端无更新接口，暂空置 */}
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" disabled>
          保存更改
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <div className="lg:col-span-4 space-y-6">
          <Card className="border-slate-200 overflow-hidden">
            <div className="h-32 bg-[#0f172a] relative">
              <div className="absolute -bottom-12 left-6">
                <div className="w-24 h-24 rounded-2xl border-4 border-white bg-slate-200 overflow-hidden shadow-lg">
                  {userInfo.avatar ? (
                    <img src={userInfo.avatar} alt="Avatar" className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-slate-400">
                      <User className="w-10 h-10" />
                    </div>
                  )}
                </div>
              </div>
            </div>
            <CardContent className="pt-16 pb-6">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-xl font-bold text-slate-900">{userInfo.name || '管理员'}</h3>
                  <p className="text-sm text-slate-500">
                    {userInfo.role === 2 ? '管理员' : userInfo.role === 1 ? '商家/主播' : '普通用户'}
                  </p>
                </div>
                <Badge className="bg-amber-100 text-amber-700 border-amber-200">在线</Badge>
              </div>
              <div className="mt-6 space-y-3">
                <div className="flex items-center gap-3 text-sm text-slate-600">
                  <Mail className="w-4 h-4 text-slate-400" />
                  <span>{userInfo.email || '未设置邮箱'}</span>
                </div>
                <div className="flex items-center gap-3 text-sm text-slate-600">
                  <Phone className="w-4 h-4 text-slate-400" />
                  <span>{userInfo.phone || '未设置手机'}</span>
                </div>
                <div className="flex items-center gap-3 text-sm text-slate-600">
                  <ShieldCheck className="w-4 h-4 text-emerald-500" />
                  <span>已认证</span>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">安全评分</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-center py-4">
                <div className="relative w-32 h-32 flex items-center justify-center">
                  <svg className="w-full h-full -rotate-90">
                    <circle cx="64" cy="64" r="58" stroke="currentColor" strokeWidth="8" fill="transparent" className="text-slate-100" />
                    <circle cx="64" cy="64" r="58" stroke="currentColor" strokeWidth="8" fill="transparent" strokeDasharray="364" strokeDashoffset="54" className="text-amber-500" />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center">
                    <span className="text-2xl font-bold text-slate-900">85</span>
                    <span className="text-xs text-slate-500">良好</span>
                  </div>
                </div>
              </div>
              <p className="text-xs text-slate-500 text-center mt-2">您的账号安全状况良好，建议定期更换密码。</p>
            </CardContent>
          </Card>
        </div>

        <div className="lg:col-span-8 space-y-6">
          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">基本资料</CardTitle>
              <CardDescription>以下信息来自系统，如需修改请联系管理员</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">姓名</label>
                  <Input
                    value={userInfo.name || ''}
                    className="bg-slate-50 border-slate-200"
                    disabled
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">角色</label>
                  <Input
                    value={userInfo.role === 2 ? '管理员' : userInfo.role === 1 ? '商家/主播' : '普通用户'}
                    className="bg-slate-50 border-slate-200"
                    disabled
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">电子邮箱</label>
                <Input
                  value={userInfo.email || ''}
                  className="bg-slate-50 border-slate-200"
                  disabled
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">手机号码</label>
                <Input
                  value={userInfo.phone || ''}
                  className="bg-slate-50 border-slate-200"
                  disabled
                />
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader>
              <CardTitle className="text-lg">安全设置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* 修改密码 - 后端无接口，暂空置 */}
              <SecuritySettingItem
                icon={Key}
                title="账号密码"
                description="定期更换密码可提高账号安全性"
                action="修改"
                disabled
              />
              {/* 两步验证 - 后端无接口，暂空置 */}
              <SecuritySettingItem
                icon={Shield}
                title="两步验证"
                description="登录时需要手机短信或 App 验证码"
                action="设置"
                disabled
              />
              {/* 消息通知设置 - 后端有部分接口 */}
              <SecuritySettingItem
                icon={Bell}
                title="消息通知"
                description="设置接收系统消息、订单提醒的方式"
                action="设置"
                disabled
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function SecuritySettingItem({ icon: Icon, title, description, action, disabled }: any) {
  return (
    <div className="flex items-center justify-between py-4 border-b border-slate-50 last:border-0">
      <div className="flex items-center gap-4">
        <div className="p-2 rounded-lg bg-slate-50 text-slate-400">
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-900">{title}</p>
          <p className="text-xs text-slate-500 mt-1">{description}</p>
        </div>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="border-slate-200"
        disabled={disabled}
      >
        {action}
      </Button>
    </div>
  )
}
