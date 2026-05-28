# 关注功能集成指南

本文档说明如何在直播间页面集成关注功能。

## 前置准备

确保已完成以下工作：
- ✅ T011: FollowButton组件已创建 (`src/components/FollowButton.tsx`)
- ✅ T012: 关注列表页面已创建 (`src/pages/Follow/index.tsx`)
- ✅ T013: 关注列表路由已添加 (`/follow`)
- ✅ T014: 关注API方法已实现 (`src/services/api.ts`)

## 集成步骤

### 1. 在直播间页面添加关注按钮

**文件**: `src/pages/Live/index.tsx`

#### 步骤 1.1: 添加导入

在文件顶部添加：

```typescript
import FollowButton from '../../components/FollowButton';
import { followApi } from '../../services/api';
```

#### 步骤 1.2: 添加关注状态

在组件状态中添加：

```typescript
const [isFollowed, setIsFollowed] = useState(false);
const [followerCount, setFollowerCount] = useState(0);
```

#### 步骤 1.3: 加载关注状态

在 `useEffect` 中加载直播间的关注状态：

```typescript
useEffect(() => {
  // 加载直播间数据时，同时加载关注状态
  const loadFollowStatus = async () => {
    if (liveRoom?.id) {
      try {
        const stats = await followApi.getFollowersStats(liveRoom.id);
        if (stats && stats.data) {
          setFollowerCount(stats.data.followers_count || 0);
          // 注意：需要从用户关注列表中判断是否已关注
          // 或者后端API直接返回当前用户的关注状态
        }
      } catch (error) {
        console.error('获取关注状态失败:', error);
      }
    }
  };

  loadFollowStatus();
}, [liveRoom?.id]);
```

#### 步骤 1.4: 在直播间头部添加关注按钮

找到直播间头部区域（约在276-287行），修改为：

```typescript
{/* 直播间头部 */}
<div style={styles.liveHeader} onClick={(e) => e.stopPropagation()}>
  <img
    src={liveRoom?.avatar}
    alt="主播"
    style={styles.liveAvatar}
  />
  <div style={styles.liveInfo}>
    <div style={styles.liveTitle}>{liveRoom?.name || '竞拍直播间'}</div>
    <div style={styles.liveViewer}>🔥 {(liveRoom?.viewerCount || 0).toLocaleString()}人在看</div>
  </div>
  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
    <FollowButton
      liveStreamId={liveRoom?.id || 0}
      initialFollowed={isFollowed}
      initialCount={followerCount}
      onFollowSuccess={(followed) => {
        setIsFollowed(followed);
        setFollowerCount(followed ? followerCount + 1 : followerCount - 1);
        setToast({
          message: followed ? '关注成功！' : '已取消关注',
          type: 'success'
        });
      }}
      onFollowError={(error) => {
        setToast({
          message: error.message || '操作失败',
          type: 'error'
        });
      }}
      size="small"
    />
    <div style={styles.liveBadge}>直播中</div>
  </div>
</div>
```

### 2. 更新 LiveRoom 接口

在 `Product` 接口之前添加或更新 `LiveRoom` 接口：

```typescript
interface LiveRoom {
  id: number;
  name: string;
  anchor: string;
  avatar: string;
  viewerCount: number;
  products: Product[];
  is_followed?: boolean;      // 新增：是否已关注
  followers_count?: number;   // 新增：关注数量
}
```

### 3. 处理关注回调

添加关注成功/失败的处理函数：

```typescript
const handleFollowSuccess = (followed: boolean) => {
  setIsFollowed(followed);
  setFollowerCount(followed ? followerCount + 1 : followerCount - 1);

  // 显示提示
  setToast({
    message: followed ? '关注成功！新商品发布时会通知您' : '已取消关注',
    type: 'success'
  });

  // 3秒后清除提示
  setTimeout(() => setToast(null), 3000);
};

const handleFollowError = (error: Error) => {
  setToast({
    message: error.message || '操作失败，请重试',
    type: 'error'
  });
  setTimeout(() => setToast(null), 3000);
};
```

## 完整示例代码片段

```typescript
// 在 LiveAuctionPage 组件中
import FollowButton from '../../components/FollowButton';
import { followApi } from '../../services/api';

const LiveAuctionPage: React.FC = () => {
  // 现有状态...
  const [isFollowed, setIsFollowed] = useState(false);
  const [followerCount, setFollowerCount] = useState(0);

  // 加载关注状态
  useEffect(() => {
    if (liveRoom?.id) {
      // 从API获取关注状态
      loadFollowStatus(liveRoom.id);
    }
  }, [liveRoom?.id]);

  const loadFollowStatus = async (liveStreamId: number) => {
    try {
      const stats = await followApi.getFollowersStats(liveStreamId);
      if (stats?.data) {
        setFollowerCount(stats.data.followers_count || 0);
        // 注意：需要在stats中返回当前用户是否已关注
        // 或者单独查询用户关注列表
      }
    } catch (error) {
      console.error('加载关注状态失败:', error);
    }
  };

  return (
    <div style={styles.container}>
      {/* 直播间背景 */}
      {/* ... */}

      {/* 直播间头部 */}
      <div style={styles.liveHeader}>
        <img src={liveRoom?.avatar} alt="主播" style={styles.liveAvatar} />
        <div style={styles.liveInfo}>
          <div style={styles.liveTitle}>{liveRoom?.name || '竞拍直播间'}</div>
          <div style={styles.liveViewer}>🔥 {liveRoom?.viewerCount || 0}人在看</div>
        </div>

        {/* 关注按钮 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <FollowButton
            liveStreamId={liveRoom?.id || 0}
            initialFollowed={isFollowed}
            initialCount={followerCount}
            onFollowSuccess={handleFollowSuccess}
            onFollowError={handleFollowError}
            size="small"
          />
          <div style={styles.liveBadge}>直播中</div>
        </div>
      </div>

      {/* 其他内容... */}
    </div>
  );
};
```

## 样式调整

如果需要调整关注按钮在直播间头部的样式，可以修改 `liveHeader` 样式：

```typescript
liveHeader: {
  display: 'flex',
  alignItems: 'center',
  padding: '12px 16px',
  backgroundColor: 'rgba(0, 0, 0, 0.3)',
  backdropFilter: 'blur(10px)',
},
```

## 测试清单

集成完成后，请测试以下场景：

- [ ] 未登录用户点击关注按钮 → 提示"请先登录"并跳转登录页
- [ ] 已登录用户点击关注 → 按钮状态立即改变（乐观更新）
- [ ] 关注成功 → 显示成功提示，关注数+1
- [ ] 关注失败 → 按钮状态回滚，显示错误提示
- [ ] 取消关注 → 按钮状态立即改变，关注数-1
- [ ] 进入直播间时正确显示关注状态和关注数
- [ ] 访问 `/follow` 路由查看关注列表

## 注意事项

1. **认证状态**: FollowButton组件内部已处理未登录状态，会自动跳转登录页
2. **乐观更新**: 按钮状态会立即改变，如果API调用失败会自动回滚
3. **后端支持**: 确保后端API `/live-streams/:id/followers/stats` 返回当前用户的关注状态
4. **性能优化**: 避免在每次渲染时都加载关注状态，应在直播间ID变化时加载

## 后续优化建议

1. **WebSocket实时更新**: 当其他用户关注/取消关注时，实时更新关注数量
2. **通知徽标**: 在关注按钮上显示未读通知数量
3. **批量关注**: 支持批量关注多个直播间
4. **关注分组**: 支持将关注的直播间分组管理
