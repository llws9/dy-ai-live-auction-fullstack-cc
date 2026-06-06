import { LiveListView } from "./LiveList"
import { useNavigate } from "react-router-dom"
import { liveStreamApi } from "@/shared/api"
import { useAuth } from "@/shared/auth"

export default function MerchantLiveList() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const displayName = user?.name || "商家用户"

  const handleCreateLiveStream = async () => {
    const liveStream = await liveStreamApi.create({
      name: `${displayName}的直播间`,
      description: `${displayName}的直播间排期`,
      streamer_name: displayName,
    })
    if (liveStream?.id) {
      navigate(`/live/detail?id=${liveStream.id}`)
    }
  }

  return (
    <LiveListView
      title="我的直播间"
      description="管理我的直播间和直播排期"
      showCreateActions
      onCreateLiveStream={handleCreateLiveStream}
    />
  )
}
