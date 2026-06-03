import React from "react"
import { Video, Users, MessageSquare, Play, Plus, MoreVertical, Loader2 } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { useNavigate } from "react-router-dom"
import { cn } from "@/lib/utils"
import { liveStreamApi } from "@/shared/api"

const statusMap: Record<number, { label: string; badgeClass: string }> = {
  0: { label: "未开播", badgeClass: "secondary" },
  1: { label: "直播中", badgeClass: "bg-rose-500 animate-pulse border-none" },
  2: { label: "已结束", badgeClass: "outline" },
}

export default function LiveList() {
  const navigate = useNavigate()
  const [liveStreams, setLiveStreams] = React.useState<any[]>([])
  const [loading, setLoading] = React.useState(true)
  const [page, setPage] = React.useState(1)
  const pageSize = 20

  // 获取直播间列表
  React.useEffect(() => {
    const fetchLiveStreams = async () => {
      setLoading(true)
      try {
        const response = await liveStreamApi.adminList({ page, page_size: pageSize })
        setLiveStreams(response.list || [])
      } catch (e) {
        console.error('获取直播间列表失败:', e)
      } finally {
        setLoading(false)
      }
    }
    fetchLiveStreams()
  }, [page])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">直播间管理</h1>
          <p className="text-sm text-slate-500">管理您的直播间和直播排期</p>
        </div>
        {/* 创建直播间 - 后端无接口，暂空置 */}
        <Button className="bg-amber-500 hover:bg-amber-600 text-[#0f172a]" disabled>
          <Plus className="mr-2 w-4 h-4" />
          创建直播间
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center min-h-[200px]">
          <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
        </div>
      ) : liveStreams.length === 0 ? (
        <div className="flex flex-col items-center justify-center min-h-[200px] text-slate-500">
          <Video className="w-12 h-12 mb-4" />
          <p>暂无直播间数据</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {liveStreams.map((stream) => {
            const statusInfo = statusMap[stream.status] || statusMap[0]
            return (
              <Card key={stream.id} className="overflow-hidden border-slate-200 group hover:shadow-xl transition-all">
                <div className="aspect-video relative overflow-hidden bg-slate-100">
                  {stream.streamer_avatar ? (
                    <img
                      src={stream.streamer_avatar}
                      alt={stream.name}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-slate-400">
                      <Video className="w-12 h-12" />
                    </div>
                  )}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent"></div>
                  <div className="absolute top-3 left-3">
                    <Badge className={statusInfo.badgeClass}>
                      {stream.status === 1 ? "● " : ""}{statusInfo.label}
                    </Badge>
                  </div>
                  <div className="absolute bottom-3 left-3 right-3 flex items-center justify-between text-white">
                    <div className="flex items-center gap-3">
                      <div className="flex items-center gap-1">
                        <Users className="w-3 h-3 text-slate-300" />
                        <span className="text-xs">{stream.viewer_count || 0}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <MessageSquare className="w-3 h-3 text-slate-300" />
                        <span className="text-xs">{stream.auction_count || 0} 场竞拍</span>
                      </div>
                    </div>
                    <span className="text-xs text-slate-300">ID: {stream.id}</span>
                  </div>
                </div>
                <CardContent className="p-5">
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="font-bold text-slate-900 group-hover:text-amber-600 transition-colors">
                        {stream.name}
                      </h3>
                      <p className="text-sm text-slate-500 mt-1">
                        主播：{stream.streamer_name || `用户#${stream.streamer_id}`}
                      </p>
                    </div>
                    <Button variant="ghost" size="icon" className="text-slate-400">
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </div>
                  <div className="mt-6 flex items-center gap-3">
                    <Button
                      className={cn(
                        "flex-1 font-bold",
                        stream.status === 1
                          ? "bg-amber-500 hover:bg-amber-600 text-[#0f172a]"
                          : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                      )}
                      onClick={() => navigate(`/live/detail?id=${stream.id}`)}
                    >
                      {stream.status === 1 ? <Play className="mr-2 w-4 h-4 fill-current" /> : null}
                      {stream.status === 1 ? "进入控制台" : "查看详情"}
                    </Button>
                    {/* 编辑直播间 - 后端无接口，暂空置 */}
                    <Button variant="outline" className="border-slate-200" disabled>
                      编辑
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}

          {/* 创建新直播间卡片 - 后端无接口，暂空置 */}
          <button
            className="aspect-video rounded-xl border-2 border-dashed border-slate-200 bg-slate-50 flex flex-col items-center justify-center gap-3 group hover:border-amber-400 transition-colors cursor-not-allowed opacity-50"
            disabled
          >
            <div className="w-12 h-12 rounded-full bg-slate-200 flex items-center justify-center text-slate-400 group-hover:bg-amber-100 group-hover:text-amber-600 transition-colors">
              <Plus className="w-6 h-6" />
            </div>
            <p className="text-sm font-semibold text-slate-900">创建新直播间</p>
            <p className="text-xs text-slate-400">(功能暂未开放)</p>
          </button>
        </div>
      )}

      {/* 分页 */}
      {liveStreams.length > 0 && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            上一页
          </Button>
          <span className="text-sm text-slate-500">第 {page} 页</span>
          <Button
            variant="outline"
            size="sm"
            disabled={liveStreams.length < pageSize}
            onClick={() => setPage(page + 1)}
          >
            下一页
          </Button>
        </div>
      )}
    </div>
  )
}
