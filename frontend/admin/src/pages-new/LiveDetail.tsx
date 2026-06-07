import React from "react"
import { ArrowLeft, Video, Loader2 } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { liveStreamApi } from "@/shared/api"
import { useAuth } from "@/shared/auth"
import { ADMIN_ROLE, MERCHANT_ROLE } from "@/shared/auth/roles"

const liveStatusLabels: Record<number, string> = {
  0: "未开播",
  1: "直播中",
  2: "已结束",
  3: "已封禁",
}

export default function LiveDetail() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { user } = useAuth()
  const liveStreamId = searchParams.get('id')

  const [loading, setLoading] = React.useState(true)
  const [submitting, setSubmitting] = React.useState(false)
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
        const data = await liveStreamApi.adminGet(Number(liveStreamId))
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

  const handleStart = async () => {
    if (!liveStreamId || submitting || liveStream?.status !== 0) return
    const confirmed = window.confirm(
      "确认开始直播？\n当前版本将由 PC 管理端发起直播状态，用于演示观看、竞拍和一口价交易链路；真实移动端推流将在后续版本接入。"
    )
    if (!confirmed) return

    setSubmitting(true)
    try {
      await liveStreamApi.start(Number(liveStreamId))
      setLiveStream((prev: any) => ({ ...prev, status: 1 }))
      alert("直播已开始")
    } catch (e) {
      console.error("开始直播失败:", e)
      alert("开始直播失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleEnd = async () => {
    if (!liveStreamId || submitting || liveStream?.status !== 1) return
    const message = isPlatformAdmin ? "确认强制结束该直播间？" : "确认结束当前直播？"
    if (!window.confirm(message)) return
    setSubmitting(true)
    try {
      const data = isPlatformAdmin
        ? await liveStreamApi.adminEnd(Number(liveStreamId))
        : await liveStreamApi.end(Number(liveStreamId))
      setLiveStream((prev: any) => ({ ...prev, status: data?.status ?? 2 }))
      alert("直播间已结束")
    } catch (e) {
      console.error("结束直播失败:", e)
      alert("结束直播失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleBan = async () => {
    if (!liveStreamId) return
    const reason = window.prompt("请输入封禁原因")
    if (!reason) return
    try {
      const data = await liveStreamApi.ban(Number(liveStreamId), reason)
      setLiveStream((prev: any) => ({ ...prev, status: data?.status ?? 3, ban_reason: data?.ban_reason ?? reason }))
      alert("直播间已封禁")
    } catch (e) {
      console.error("封禁直播失败:", e)
      alert("封禁直播失败")
    }
  }

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
  const canStartLive = liveStream.status === 0
  const statusLabel = liveStatusLabels[liveStream.status] || "未知状态"
  const isMerchant = user?.role === MERCHANT_ROLE
  const isPlatformAdmin = user?.role === ADMIN_ROLE

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
                  <p className="text-xl font-bold tracking-widest text-slate-400">直播间{statusLabel}</p>
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
                <span className="font-bold">{statusLabel}</span>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200">
            <CardHeader><CardTitle className="text-lg">操作中心</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {isMerchant && (
                <>
                  <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm leading-6 text-amber-900">
                    当前版本支持通过 PC 管理端发起直播状态，用于商品讲解、竞拍和一口价交易链路演示；移动端主播推流能力将在后续版本接入。
                  </div>
                  {isLive ? (
                    <Button
                      variant="destructive"
                      className="w-full"
                      onClick={handleEnd}
                      disabled={submitting}
                    >
                      {submitting ? "结束中..." : "结束直播"}
                    </Button>
                  ) : (
                    <Button
                      className="w-full bg-amber-500 text-[#0f172a] hover:bg-amber-600"
                      onClick={handleStart}
                      disabled={submitting || !canStartLive}
                    >
                      <Video className="mr-2 w-4 h-4" />
                      {canStartLive ? (submitting ? "开始中..." : "开始直播") : statusLabel}
                    </Button>
                  )}
                </>
              )}
              {isPlatformAdmin && (
                <>
                  <Button variant="outline" className="w-full" onClick={handleBan}>
                    封禁直播间
                  </Button>
                  <Button variant="destructive" className="w-full" onClick={handleEnd} disabled={submitting || liveStream.status !== 1}>
                    {submitting ? "关闭中..." : "关闭直播"}
                  </Button>
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
