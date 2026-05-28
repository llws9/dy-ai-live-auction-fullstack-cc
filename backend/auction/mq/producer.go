package mq

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

// NotificationProducer 通知消息生产者
type NotificationProducer struct {
	channel *amqp.Channel
}

// NewNotificationProducer 创建生产者
func NewNotificationProducer(rmq *RabbitMQConnection) *NotificationProducer {
	return &NotificationProducer{
		channel: rmq.GetChannel(),
	}
}

// SendNewProductNotification 发送新商品发布通知（立即发送）
func (p *NotificationProducer) SendNewProductNotification(
	liveStreamID int64,
	productID int64,
	productName string,
	creatorName string,
) error {
	msg := NotificationMessage{
		LiveStreamID: liveStreamID,
		Type:         "new_product",
		ProductID:    productID,
		ProductName:  productName,
		CreatorName:  creatorName,
		CreatedAt:    time.Now(),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		context.Background(),
		"notification.main", // exchange
		"new_product",       // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // 持久化
		},
	)
}

// SendAuctionStartingNotification 发送竞拍开始通知（延迟30分钟）
// 使用 TTL + DLX 实现延迟队列：消息在延迟队列中等待TTL时间后，转发到就绪队列
func (p *NotificationProducer) SendAuctionStartingNotification(
	liveStreamID int64,
	auctionID int64,
	productID int64,
	productName string,
	startTime time.Time,
) error {
	// 计算延迟时间 = 开始时间 - 30分钟 - 当前时间
	delayMs := startTime.Add(-30 * time.Minute).Sub(time.Now()).Milliseconds()
	if delayMs < 0 {
		delayMs = 0 // 如果已经过了30分钟前，立即发送到就绪队列
	}

	msg := NotificationMessage{
		LiveStreamID: liveStreamID,
		Type:         "auction_starting",
		ProductID:    productID,
		AuctionID:    auctionID,
		ProductName:  productName,
		StartTime:    startTime,
		CreatedAt:    time.Now(),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 直接发送到就绪队列（如果不需要延迟）
	if delayMs == 0 {
		return p.channel.PublishWithContext(
			context.Background(),
			"notification.main",        // exchange
			"auction_starting_ready",   // routing key
			false,                      // mandatory
			false,                      // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent,
			},
		)
	}

	// 发送到延迟队列，设置消息级别的TTL
	// 消息过期后自动转发到 notification.main 交换机，routing key 为 auction_starting_ready
	return p.channel.PublishWithContext(
		context.Background(),
		"",                                     // exchange（空字符串表示直接发送到队列）
		"notification.auction_starting_delayed", // routing key（队列名）
		false,                                  // mandatory
		false,                                  // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Expiration:   string(delayMs), // 消息TTL（毫秒）
		},
	)
}

// SendProductUnpublishedNotification 发送商品下架通知（立即发送）
func (p *NotificationProducer) SendProductUnpublishedNotification(
	liveStreamID int64,
	productID int64,
	productName string,
	reason string,
) error {
	msg := NotificationMessage{
		LiveStreamID: liveStreamID,
		Type:         "product_unpublished",
		ProductID:    productID,
		ProductName:  productName,
		Reason:       reason,
		CreatedAt:    time.Now(),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		context.Background(),
		"notification.main",    // exchange
		"product_unpublished",  // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

// SendAuctionEndedNotification 发送竞拍结束通知（立即发送）
func (p *NotificationProducer) SendAuctionEndedNotification(
	liveStreamID int64,
	auctionID int64,
	productID int64,
	productName string,
	winnerID int64,
	winnerName string,
	finalPrice float64,
) error {
	msg := NotificationMessage{
		LiveStreamID: liveStreamID,
		Type:         "auction_ended",
		ProductID:    productID,
		AuctionID:    auctionID,
		ProductName:  productName,
		WinnerID:     winnerID,
		WinnerName:   winnerName,
		FinalPrice:   finalPrice,
		CreatedAt:    time.Now(),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		context.Background(),
		"notification.main", // exchange
		"auction_ended",     // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}
