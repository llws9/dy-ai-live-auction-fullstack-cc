import React from "react"
import { useNavigate } from "react-router-dom"
import { Gavel, Lock, ArrowRight, Mail, Phone } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { authApi, LoginRequest, useAuth } from "@/shared/auth"

export default function Login() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [loginType, setLoginType] = React.useState<'email' | 'phone'>('email')
  const [formData, setFormData] = React.useState({
    email: '',
    phone: '',
    password: ''
  })

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
    setError(null)
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const loginData: LoginRequest = {
        password: formData.password,
        ...(loginType === 'email' ? { email: formData.email } : { phone: formData.phone })
      }

      const response = await authApi.login(loginData)

      login(response.token, response.user)

      // 跳转到首页
      navigate("/dashboard")
    } catch (err: any) {
      setError(err.message || '登录失败，请检查账号密码')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-[#0f172a] relative overflow-hidden">
      {/* Background Decorative Elements */}
      <div className="absolute top-[-10%] right-[-10%] w-[40%] h-[40%] bg-amber-500/10 rounded-full blur-[120px]"></div>
      <div className="absolute bottom-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-500/10 rounded-full blur-[120px]"></div>

      <div className="w-full max-w-md px-4 z-10">
        <div className="flex flex-col items-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-amber-500 flex items-center justify-center mb-4 shadow-2xl shadow-amber-500/20">
            <Gavel className="w-10 h-10 text-[#0f172a]" />
          </div>
          <h1 className="text-3xl font-bold text-white tracking-tight">直播竞拍后台系统</h1>
          <p className="text-slate-400 mt-2 font-medium">高端典雅 · 专业高效</p>
        </div>

        <Card className="border-slate-800 bg-slate-900/50 backdrop-blur-xl text-white shadow-2xl">
          <CardHeader className="space-y-1">
            <CardTitle className="text-2xl text-center">欢迎回来</CardTitle>
            <CardDescription className="text-slate-400 text-center">
              请输入您的账号和密码进行登录
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleLogin} className="space-y-4">
              {/* 登录方式切换 */}
              <div className="flex gap-2 mb-4">
                <button
                  type="button"
                  onClick={() => setLoginType('email')}
                  className={`flex-1 py-2 text-sm rounded-lg transition-colors ${
                    loginType === 'email'
                      ? 'bg-amber-500/20 text-amber-500 border border-amber-500'
                      : 'bg-slate-800 text-slate-400 border border-slate-700'
                  }`}
                >
                  邮箱登录
                </button>
                <button
                  type="button"
                  onClick={() => setLoginType('phone')}
                  className={`flex-1 py-2 text-sm rounded-lg transition-colors ${
                    loginType === 'phone'
                      ? 'bg-amber-500/20 text-amber-500 border border-amber-500'
                      : 'bg-slate-800 text-slate-400 border border-slate-700'
                  }`}
                >
                  手机登录
                </button>
              </div>

              {/* 账号输入 */}
              <div className="space-y-2">
                <div className="relative">
                  {loginType === 'email'
                    ? <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                    : <Phone className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                  }
                  <Input
                    type={loginType === 'email' ? 'email' : 'tel'}
                    placeholder={loginType === 'email' ? '邮箱地址' : '手机号码'}
                    className="pl-10 bg-slate-950/50 border-slate-700 text-white placeholder:text-slate-500 focus:ring-amber-500"
                    value={loginType === 'email' ? formData.email : formData.phone}
                    onChange={(e) => handleInputChange(loginType, e.target.value)}
                    required
                  />
                </div>
              </div>

              {/* 密码输入 */}
              <div className="space-y-2">
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
                  <Input
                    type="password"
                    placeholder="密码"
                    className="pl-10 bg-slate-950/50 border-slate-700 text-white placeholder:text-slate-500 focus:ring-amber-500"
                    value={formData.password}
                    onChange={(e) => handleInputChange('password', e.target.value)}
                    required
                  />
                </div>
              </div>

              {/* 错误提示 */}
              {error && (
                <div className="text-red-400 text-sm text-center py-2 bg-red-500/10 rounded-lg">
                  {error}
                </div>
              )}

              <Button
                type="submit"
                className="w-full bg-amber-500 hover:bg-amber-600 text-[#0f172a] font-bold h-12 text-lg transition-all group"
                disabled={loading}
              >
                {loading ? "登录中..." : "立即登录"}
                <ArrowRight className="ml-2 w-5 h-5 group-hover:translate-x-1 transition-transform" />
              </Button>
            </form>

            <div className="mt-6 text-center">
              <p className="text-sm text-slate-500">
                忘记密码？请联系系统管理员
              </p>
            </div>
          </CardContent>
        </Card>

        <div className="mt-8 text-center">
          <p className="text-slate-500 text-xs">
            © 2026 直播竞拍管理系统. All Rights Reserved.
          </p>
        </div>
      </div>
    </div>
  )
}
