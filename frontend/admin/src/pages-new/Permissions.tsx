import React from "react"
import { Shield, Users, Lock, MoreHorizontal, Plus, ShieldCheck } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"

const roles = [
  { id: 1, name: "超级管理员", code: "SUPER_ADMIN", users: 2, description: "拥有系统所有操作权限" },
  { id: 2, name: "商家运营", code: "MERCHANT_OPERATOR", users: 15, description: "管理商品、直播间与订单" },
  { id: 3, name: "财务审核", code: "FINANCE_AUDITOR", users: 5, description: "查看财务统计与提现审批" },
  { id: 4, name: "客服人员", code: "CUSTOMER_SERVICE", users: 20, description: "处理用户咨询与投诉" },
]

const adminUsers = [
  { id: 1, name: "王经理", role: "高级运营", email: "wang@auction.com", status: "active" },
  { id: 2, name: "李主管", role: "财务经理", email: "lee@auction.com", status: "active" },
  { id: 3, name: "张小美", role: "金牌客服", email: "zhang@auction.com", status: "active" },
  { id: 4, name: "赵铁柱", role: "系统管理员", email: "zhao@auction.com", status: "inactive" },
]

export default function Permissions() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">权限管理</h1>
          <p className="text-sm text-slate-500">配置角色权限与管理后台用户</p>
        </div>
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]">
          <Plus className="mr-2 w-4 h-4" />
          新增角色/用户
        </Button>
      </div>

      <Tabs defaultValue="roles" className="space-y-6">
        <TabsList className="bg-white border border-slate-200 p-1 h-12">
          <TabsTrigger value="roles" className="px-8 h-10 data-[state=active]:bg-amber-500 data-[state=active]:text-[#0f172a]">角色管理</TabsTrigger>
          <TabsTrigger value="users" className="px-8 h-10 data-[state=active]:bg-amber-500 data-[state=active]:text-[#0f172a]">用户管理</TabsTrigger>
        </TabsList>

        <TabsContent value="roles" className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {roles.map((role) => (
            <Card key={role.id} className="border-slate-200 hover:border-amber-400 transition-all group">
              <CardHeader className="flex flex-row items-start justify-between">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 rounded-xl bg-slate-100 flex items-center justify-center text-slate-400 group-hover:bg-amber-100 group-hover:text-amber-600 transition-colors">
                    <Shield className="w-6 h-6" />
                  </div>
                  <div>
                    <CardTitle className="text-lg">{role.name}</CardTitle>
                    <CardDescription className="text-xs font-mono mt-1">{role.code}</CardDescription>
                  </div>
                </div>
                <Button variant="ghost" size="icon" className="text-slate-400">
                  <MoreHorizontal className="w-4 h-4" />
                </Button>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-slate-600 line-clamp-2 min-h-[40px]">{role.description}</p>
                <div className="mt-6 flex items-center justify-between">
                  <div className="flex items-center gap-2 text-xs text-slate-500">
                    <Users className="w-3 h-3" />
                    <span>{role.users} 个关联用户</span>
                  </div>
                  <Button variant="outline" size="sm" className="border-slate-200 text-slate-600">
                    编辑权限
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="users">
          <Card className="border-slate-200">
            <CardContent className="p-0">
              <Table>
                <TableHeader className="bg-slate-50/50">
                  <TableRow>
                    <TableHead>用户名</TableHead>
                    <TableHead>所属角色</TableHead>
                    <TableHead>电子邮箱</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {adminUsers.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium text-slate-900">{user.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-normal border-slate-200">{user.role}</Badge>
                      </TableCell>
                      <TableCell className="text-slate-500">{user.email}</TableCell>
                      <TableCell>
                        {user.status === "active" ? (
                          <Badge variant="success" className="bg-emerald-50 text-emerald-700 border-emerald-200">正常</Badge>
                        ) : (
                          <Badge variant="outline" className="text-slate-400">已禁用</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm" className="text-amber-600 hover:text-amber-700">编辑</Button>
                        <Button variant="ghost" size="sm" className="text-slate-400">重置密码</Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
