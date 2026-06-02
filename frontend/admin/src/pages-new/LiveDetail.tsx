import React from "react"
import { ArrowLeft, Video, Loader2 } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { liveStreamApi } from "@/shared/api"

export default function LiveDetail() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const liveStreamId = searchParams.get('id')

  const [loading, setLoading] = React.useState(true)
  const [liveStream, setLiveStream] = React.useState<any>(null)

  // 获取直播间详情
  React.useEffect(() => {
    if (!liveStreamId) {
      navigate('/live/list')
      return
    }

    const fetchLiveStream = async () => {
      setLoading(true)
      try {
        const data = await liveStreamApi.get(Number(liveStreamId))
        setLiveStream(data)
      } catch (e) {
        console.error('获取直播间详情失败:', e)
        alert('获取直播间详情失败')
        navigate('/live/list')
      } finally {
        setLoading(false)
      }
    }
    fetchLiveStream()
  }, [liveStreamId, navigate])

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-400" />
      </div>
    )
  }

  if (!liveStream) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-slate-500">直播间不存在</p>
      </div>
    )
  }

  const isLive = liveStream.status === 1

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" onClick={() => navigate("/live/list")} className="border-slate-200">
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-slate-900">直播间控制台</h1>
          <p className="text-sm text-slate-500">{liveStream.name} | 主播: {liveStream.streamer_name || `用户#${liveStream.streamer_id}`}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="lg:col-span-2 aspect-video bg-black flex items-center justify-center text-white relative overflow-hidden">
          {liveStream.streamer_avatar ? (
            <img src={liveStream.streamer_avatar} alt={liveStream.name} className="w-full h-full object-cover opacity-50" />
          ) : (
            <div className="w-full h-full bg-slate-900" />
          )}
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center">
              {isLive ? (
                <>
                  <Video className="w-16 h-16 mx-auto mb-4 text-rose-500 animate-pulse" />
                  <p className="text-xl font-bold tracking-widest">正在直播中...</p>
                </>
              ) : (
                <>
                  <Video className="w-16 h-16 mx-auto mb-4 text-slate-400" />
                  <p className="text-xl font-bold tracking-widest text-slate-400">直播间未开播</p>
                </>
              )}
            </div>
          </div>
        </Card>

        <div className="space-y-6">
          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">实时数据</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between border-b pb-2">
                <span className="text-slate-500">在线人数</span>
                <span className="font-bold">{liveStream.viewer_count || 0}</span>
              </div>
              <div className="flex justify-between border-b pb-2">
                <span className="text-slate-500">竞拍场次</span>
                <span className="font-bold">{liveStream.auction_count || 0}</span>
              </div>
              <div className="flex justify-between border-b pb-2">
                <span className="text-slate-500">直播间状态</span>
                <span className="font-bold">{isLive ? '直播中' : '未开播'}</span>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">操作中心</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {/* 推送商品 - 后端无接口，暂空置 */}
              <Button className="w-full bg-amber-500 text-[#0f172a]" disabled>
                推送商品
              </Button>
              {/* 静音/禁言 - 后端无接口，暂空置 */}
              <Button variant="outline" className="w-full" disabled>
                静音/禁言
              </Button>
              {/* 关闭直播 - 后端无接口，暂空置 */}
              <Button variant="destructive" className="w-full" disabled>
                关闭直播
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
