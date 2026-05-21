package websocket

// MessageType WebSocket 消息类型
type MessageType string

const (
	// 客户端 -> 服务端
	MessageTypePing MessageType = "ping"

	// 服务端 -> 客户端
	MessageTypePong          MessageType = "pong"
	MessageTypeBidPlaced     MessageType = "bid_placed"
	MessageTypeRankUpdate    MessageType = "rank_update"
	MessageTypeOvertaken     MessageType = "overtaken"
	MessageTypeDelayTriggered MessageType = "delay_triggered"
	MessageTypeAuctionEnded  MessageType = "auction_ended"
	MessageTypeTimeSync      MessageType = "time_sync"
	MessageTypeError         MessageType = "error"
)

// Message WebSocket 消息基础结构
type Message struct {
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// BidPlacedData 出价通知数据
type BidPlacedData struct {
	AuctionID   int64   `json:"auction_id"`
	UserID      int64   `json:"user_id"`
	UserName    string  `json:"user_name,omitempty"`
	Amount      float64 `json:"amount"`
	CurrentPrice float64 `json:"current_price"`
	BidTime     int64   `json:"bid_time"`
}

// RankUpdateData 排名更新数据
type RankUpdateData struct {
	AuctionID int64        `json:"auction_id"`
	Ranking   []RankItem   `json:"ranking"`
}

// RankItem 排名项
type RankItem struct {
	Rank     int     `json:"rank"`
	UserID   int64   `json:"user_id"`
	UserName string  `json:"user_name,omitempty"`
	Amount   float64 `json:"amount"`
}

// OvertakenData 被超越通知数据
type OvertakenData struct {
	AuctionID      int64   `json:"auction_id"`
	OvertakenBy    int64   `json:"overtaken_by"`
	OvertakenName  string  `json:"overtaken_name,omitempty"`
	NewPrice       float64 `json:"new_price"`
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
	AuctionID  int64   `json:"auction_id"`
	WinnerID   int64   `json:"winner_id"`
	WinnerName string  `json:"winner_name,omitempty"`
	FinalPrice float64 `json:"final_price"`
	EndTime    int64   `json:"end_time"`
}

// TimeSyncData 时间同步数据
type TimeSyncData struct {
	ServerTime int64 `json:"server_time"`
	EndTime    int64 `json:"end_time,omitempty"`
}

// ErrorData 错误数据
type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
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
func NewTimeSyncMessage(serverTime, endTime int64) *Message {
	return NewMessage(MessageTypeTimeSync, &TimeSyncData{
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
