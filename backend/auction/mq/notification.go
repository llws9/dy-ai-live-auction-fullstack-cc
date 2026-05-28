package mq

import (
	"fmt"
	"log"
	"time"
)

// NotificationMessage 通知消息结构体
type NotificationMessage struct {
	LiveStreamID int64     `json:"live_stream_id"`
	Type         string    `json:"type"`           // new_product, auction_starting, auction_ended, product_unpublished
	ProductID    int64     `json:"product_id"`
	ProductName  string    `json:"product_name"`
	AuctionID    int64     `json:"auction_id,omitempty"`
	CreatorName  string    `json:"creator_name,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	WinnerID     int64     `json:"winner_id,omitempty"`
	WinnerName   string    `json:"winner_name,omitempty"`
	FinalPrice   float64   `json:"final_price,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	RetryCount   int       `json:"retry_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notificationService NotificationServiceInterface
}

// NotificationServiceInterface 通知服务接口
type NotificationServiceInterface interface {
	ProcessNotification(msg *NotificationMessage) error
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(svc NotificationServiceInterface) *NotificationHandler {
	return &NotificationHandler{
		notificationService: svc,
	}
}

// Handle 处理通知消息
func (h *NotificationHandler) Handle(msg *NotificationMessage) error {
	log.Printf("Handling notification: Type=%s, LiveStreamID=%d", msg.Type, msg.LiveStreamID)

	return h.notificationService.ProcessNotification(msg)
}

// GenerateTitle 生成通知标题
func (msg *NotificationMessage) GenerateTitle() string {
	switch msg.Type {
	case "new_product":
		return "新商品上架"
	case "auction_starting":
		return "竞拍即将开始"
	case "auction_ended":
		return "竞拍已结束"
	case "product_unpublished":
		return "商品已下架"
	default:
		return "通知"
	}
}

// GenerateContent 生成通知内容
func (msg *NotificationMessage) GenerateContent() string {
	switch msg.Type {
	case "new_product":
		return fmt.Sprintf("直播间发布了新商品【%s】，快来参与竞拍吧！", msg.ProductName)
	case "auction_starting":
		return fmt.Sprintf("商品【%s】的竞拍即将在30分钟后开始，不要错过！", msg.ProductName)
	case "auction_ended":
		return fmt.Sprintf("商品【%s】的竞拍已结束，中标者：%s，成交价：%.2f元", msg.ProductName, msg.WinnerName, msg.FinalPrice)
	case "product_unpublished":
		return fmt.Sprintf("商品【%s】已被商家下架，原因：%s", msg.ProductName, msg.Reason)
	default:
		return "您有一条新通知"
	}
}

// GenerateData 生成通知扩展数据
func (msg *NotificationMessage) GenerateData() map[string]interface{} {
	data := map[string]interface{}{
		"live_stream_id": msg.LiveStreamID,
		"product_id":     msg.ProductID,
	}

	if msg.AuctionID > 0 {
		data["auction_id"] = msg.AuctionID
	}

	return data
}
