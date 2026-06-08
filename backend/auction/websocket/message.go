package websocket

import (
	"time"

	"github.com/shopspring/decimal"
)

// MessageType WebSocket 消息类型
type MessageType string

const (
	// 客户端 -> 服务端
	MessageTypePing        MessageType = "ping"
	MessageTypeSyncRequest MessageType = "sync_request"

	// 服务端 -> 客户端
	MessageTypePong           MessageType = "pong"
	MessageTypeBidPlaced      MessageType = "bid_placed"
	MessageTypeRankUpdate     MessageType = "rank_update"
	MessageTypeOvertaken      MessageType = "overtaken"
	MessageTypeDelayTriggered MessageType = "delay_triggered"
	MessageTypeAuctionEnded   MessageType = "auction_ended"
	MessageTypeTimeSync       MessageType = "time_sync"
	MessageTypeSyncResponse   MessageType = "sync_response"
	MessageTypeError          MessageType = "error"
	MessageTypeNotification   MessageType = "notification" // 通知消息类型

	// 天灯相关消息类型
	MessageTypeSkyLampActivated MessageType = "sky_lamp_activated" // 天灯开启
	MessageTypeSkyLampAutoBid   MessageType = "sky_lamp_auto_bid"  // 自动跟价
	MessageTypeSkyLampStopped   MessageType = "sky_lamp_stopped"   // 天灯停止

	// 一口价秒杀相关消息类型
	MessageTypeFixedPriceListed  MessageType = "fixed_price_listed"
	MessageTypeFixedPriceStock   MessageType = "fixed_price_stock"
	MessageTypeFixedPriceSoldOut MessageType = "fixed_price_sold_out"
	MessageTypeFixedPriceOffline MessageType = "fixed_price_offline"
	MessageTypeFixedPriceFlair   MessageType = "fixed_price_flair"

	// 弹幕相关消息类型（M2）
	MessageTypeChatSend           MessageType = "chat_send"            // 客户端 -> 服务端
	MessageTypeChatMessage        MessageType = "chat_message"         // 服务端 -> 客户端
	MessageTypeLivePresenceUpdate MessageType = "live_presence_update" // 服务端 -> 客户端
)

// 弹幕错误码
const (
	ChatErrCodeLengthExceeded    = 40001
	ChatErrCodeBlockedWord       = 40002
	ChatErrCodeRateLimited       = 40003
	ChatErrCodeInvalidLiveStream = 40004
	ChatErrCodeNotAuthenticated  = 40101
)

// Message WebSocket 消息基础结构
type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// BidPlacedData 出价通知数据
type BidPlacedData struct {
	AuctionID    int64           `json:"auction_id"`
	UserID       int64           `json:"user_id"`
	UserName     string          `json:"user_name,omitempty"`
	Amount       decimal.Decimal `json:"amount"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	BidTime      int64           `json:"bid_time"`
}

// RankUpdateData 排名更新数据
type RankUpdateData struct {
	AuctionID int64      `json:"auction_id"`
	Ranking   []RankItem `json:"ranking"`
}

// RankItem 排名项
type RankItem struct {
	Rank     int             `json:"rank"`
	UserID   int64           `json:"user_id"`
	UserName string          `json:"user_name,omitempty"`
	Amount   decimal.Decimal `json:"amount"`
}

// OvertakenData 被超越通知数据
type OvertakenData struct {
	AuctionID     int64           `json:"auction_id"`
	OvertakenBy   int64           `json:"overtaken_by"`
	OvertakenName string          `json:"overtaken_name,omitempty"`
	NewPrice      decimal.Decimal `json:"new_price"`
}

// DelayTriggeredData 延时触发数据
type DelayTriggeredData struct {
	AuctionID      int64 `json:"auction_id"`
	DelayDuration  int   `json:"delay_duration"`
	NewEndTime     int64 `json:"new_end_time"`
	RemainingDelay int   `json:"remaining_delay"`
	MaxDelay       int   `json:"max_delay"`
}

// AuctionEndedData 竞拍结束数据
type AuctionEndedData struct {
	AuctionID  int64           `json:"auction_id"`
	WinnerID   int64           `json:"winner_id"`
	WinnerName string          `json:"winner_name,omitempty"`
	FinalPrice decimal.Decimal `json:"final_price"`
	EndTime    int64           `json:"end_time"`
}

// TimeSyncData 时间同步数据
type TimeSyncData struct {
	AuctionID  int64 `json:"auction_id"`
	ServerTime int64 `json:"server_time"`
	EndTime    int64 `json:"end_time,omitempty"`
}

// SyncRequestData 状态同步请求数据
type SyncRequestData struct {
	AuctionID int64 `json:"auction_id"`
}

// SyncResponseData 状态同步响应数据
type SyncResponseData struct {
	AuctionID    int64           `json:"auction_id"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	WinnerID     int64           `json:"winner_id"`
	EndTime      int64           `json:"end_time"`
	Status       int             `json:"status"`
	Ranking      []RankItem      `json:"ranking,omitempty"`
}

// ErrorData 错误数据
type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NotificationData 通知数据
type NotificationData struct {
	ID        int64                  `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt int64                  `json:"created_at"`
}

// SkyLampActivatedData 天灯开启通知
type SkyLampActivatedData struct {
	AuctionID        int64           `json:"auction_id"`
	UserID           int64           `json:"user_id"`
	SubscriptionID   int64           `json:"subscription_id"`
	InitialBidAmount decimal.Decimal `json:"initial_bid_amount"`
	MaxPriceLimit    decimal.Decimal `json:"max_price_limit"`
}

// SkyLampAutoBidData 自动跟价通知
type SkyLampAutoBidData struct {
	AuctionID       int64           `json:"auction_id"`
	UserID          int64           `json:"user_id"`
	Amount          decimal.Decimal `json:"amount"`
	RemainingBudget decimal.Decimal `json:"remaining_budget"`
	AutoBidCount    int             `json:"auto_bid_count"`
}

// SkyLampStoppedData 天灯停止通知
type SkyLampStoppedData struct {
	AuctionID     int64  `json:"auction_id"`
	UserID        int64  `json:"user_id"`
	Reason        string `json:"reason"` // "limit_reached" | "cancelled" | "auction_ended" | "max_count_reached"
	TotalBidCount int    `json:"total_bid_count"`
}

// FixedPriceListedData 一口价上架通知。
type FixedPriceListedData struct {
	ItemID         int64  `json:"item_id"`
	LiveStreamID   int64  `json:"live_stream_id"`
	ProductID      int64  `json:"product_id"`
	Price          string `json:"price"`
	TotalStock     int    `json:"total_stock"`
	RemainingStock int    `json:"remaining_stock"`
	Status         string `json:"status"`
}

// FixedPriceStockData 一口价库存变更通知。
type FixedPriceStockData struct {
	ItemID         int64 `json:"item_id"`
	RemainingStock int   `json:"remaining_stock"`
}

// FixedPriceSoldOutData 一口价售罄通知。
type FixedPriceSoldOutData struct {
	ItemID int64 `json:"item_id"`
}

// FixedPriceOfflineData 一口价下架通知。
type FixedPriceOfflineData struct {
	ItemID int64 `json:"item_id"`
}

// FixedPriceFlairData 一口价购买飘屏通知。
type FixedPriceFlairData struct {
	ItemID  int64  `json:"item_id"`
	BuyerID int64  `json:"buyer_id"`
	Price   string `json:"price"`
}

// NewMessage 创建消息
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: 0, // 将由发送时填充
		Data:      data,
	}
}

// NewPingMessage 创建 ping 消息
func NewPingMessage() *Message {
	return &Message{
		Type: MessageTypePing,
	}
}

// NewPongMessage 创建 pong 消息
func NewPongMessage() *Message {
	return &Message{
		Type:      MessageTypePong,
		Timestamp: 0,
	}
}

// NewBidPlacedMessage 创建出价通知消息
func NewBidPlacedMessage(data *BidPlacedData) *Message {
	return NewMessage(MessageTypeBidPlaced, data)
}

// NewDelayTriggeredMessage 创建延时触发消息
func NewDelayTriggeredMessage(data *DelayTriggeredData) *Message {
	return NewMessage(MessageTypeDelayTriggered, data)
}

// NewAuctionEndedMessage 创建竞拍结束消息
func NewAuctionEndedMessage(data *AuctionEndedData) *Message {
	return NewMessage(MessageTypeAuctionEnded, data)
}

// NewTimeSyncMessage 创建时间同步消息
func NewTimeSyncMessage(auctionID, serverTime, endTime int64) *Message {
	return NewMessage(MessageTypeTimeSync, &TimeSyncData{
		AuctionID:  auctionID,
		ServerTime: serverTime,
		EndTime:    endTime,
	})
}

// NewErrorMessage 创建错误消息
func NewErrorMessage(code int, message string) *Message {
	return NewMessage(MessageTypeError, &ErrorData{
		Code:    code,
		Message: message,
	})
}

// NewSyncRequestMessage 创建同步请求消息
func NewSyncRequestMessage(auctionID int64) *Message {
	return NewMessage(MessageTypeSyncRequest, &SyncRequestData{
		AuctionID: auctionID,
	})
}

// NewSyncResponseMessage 创建同步响应消息
func NewSyncResponseMessage(data *SyncResponseData) *Message {
	return NewMessage(MessageTypeSyncResponse, data)
}

// NewRankUpdateMessage 创建排名更新消息
func NewRankUpdateMessage(auctionID int64, ranking []RankItem) *Message {
	return NewMessage(MessageTypeRankUpdate, &RankUpdateData{
		AuctionID: auctionID,
		Ranking:   ranking,
	})
}

// NewNotificationMessage 创建通知消息
func NewNotificationMessage(data *NotificationData) *Message {
	return NewMessage(MessageTypeNotification, data)
}

// NewSkyLampActivatedMessage 创建天灯开启消息
func NewSkyLampActivatedMessage(auctionID, userID, subscriptionID int64, initialBidAmount, maxPriceLimit decimal.Decimal) *Message {
	return &Message{
		Type:      MessageTypeSkyLampActivated,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: SkyLampActivatedData{
			AuctionID:        auctionID,
			UserID:           userID,
			SubscriptionID:   subscriptionID,
			InitialBidAmount: initialBidAmount,
			MaxPriceLimit:    maxPriceLimit,
		},
	}
}

// NewSkyLampAutoBidMessage 创建自动跟价消息
func NewSkyLampAutoBidMessage(auctionID, userID int64, amount, remainingBudget decimal.Decimal, autoBidCount int) *Message {
	return &Message{
		Type:      MessageTypeSkyLampAutoBid,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: SkyLampAutoBidData{
			AuctionID:       auctionID,
			UserID:          userID,
			Amount:          amount,
			RemainingBudget: remainingBudget,
			AutoBidCount:    autoBidCount,
		},
	}
}

// NewSkyLampStoppedMessage 创建天灯停止消息
func NewSkyLampStoppedMessage(auctionID, userID int64, reason string, totalBidCount int) *Message {
	return &Message{
		Type:      MessageTypeSkyLampStopped,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: SkyLampStoppedData{
			AuctionID:     auctionID,
			UserID:        userID,
			Reason:        reason,
			TotalBidCount: totalBidCount,
		},
	}
}

// NewFixedPriceListedMessage 创建一口价上架消息。
func NewFixedPriceListedMessage(data *FixedPriceListedData) *Message {
	return NewMessage(MessageTypeFixedPriceListed, data)
}

// NewFixedPriceStockMessage 创建一口价库存变更消息。
func NewFixedPriceStockMessage(itemID int64, remainingStock int) *Message {
	return NewMessage(MessageTypeFixedPriceStock, &FixedPriceStockData{ItemID: itemID, RemainingStock: remainingStock})
}

// NewFixedPriceSoldOutMessage 创建一口价售罄消息。
func NewFixedPriceSoldOutMessage(itemID int64) *Message {
	return NewMessage(MessageTypeFixedPriceSoldOut, &FixedPriceSoldOutData{ItemID: itemID})
}

// NewFixedPriceOfflineMessage 创建一口价下架消息。
func NewFixedPriceOfflineMessage(itemID int64) *Message {
	return NewMessage(MessageTypeFixedPriceOffline, &FixedPriceOfflineData{ItemID: itemID})
}

// NewFixedPriceFlairMessage 创建一口价购买飘屏消息。
func NewFixedPriceFlairMessage(data *FixedPriceFlairData) *Message {
	return NewMessage(MessageTypeFixedPriceFlair, data)
}

// ChatSendData 客户端发送的弹幕请求
type ChatSendData struct {
	LiveStreamID int64  `json:"live_stream_id"`
	Text         string `json:"text"`
	ClientMsgID  string `json:"client_msg_id"`
}

// ChatMessageData 服务端广播的弹幕消息
type ChatMessageData struct {
	LiveStreamID int64  `json:"live_stream_id"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Text         string `json:"text"`
	SentAt       int64  `json:"sent_at"`
	ClientMsgID  string `json:"client_msg_id,omitempty"`
}

// NewChatMessage 创建弹幕广播消息
func NewChatMessage(data *ChatMessageData) *Message {
	return NewMessage(MessageTypeChatMessage, data)
}

// LivePresenceViewer 是在线用户头像区域展示所需的最小用户信息。
type LivePresenceViewer struct {
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// LivePresenceUpdateData 是直播间在线状态快照。
type LivePresenceUpdateData struct {
	LiveStreamID int64                `json:"live_stream_id"`
	ViewerCount  int                  `json:"viewer_count"`
	Viewers      []LivePresenceViewer `json:"viewers"`
}

// NewLivePresenceUpdateMessage 创建直播间在线状态更新消息。
func NewLivePresenceUpdateMessage(data *LivePresenceUpdateData) *Message {
	return NewMessage(MessageTypeLivePresenceUpdate, data)
}
