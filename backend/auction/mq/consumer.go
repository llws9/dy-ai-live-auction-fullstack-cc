package mq

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

// NotificationConsumer 通知消息消费者
type NotificationConsumer struct {
	channel *amqp.Channel
	handler *NotificationHandler
}

// NewNotificationConsumer 创建消费者
func NewNotificationConsumer(rmq *RabbitMQConnection, handler *NotificationHandler) *NotificationConsumer {
	// 设置 QoS（每个消费者最多处理10条未确认消息）
	rmq.GetChannel().Qos(10, 0, false)

	return &NotificationConsumer{
		channel: rmq.GetChannel(),
		handler: handler,
	}
}

// Start 启动消费者
func (c *NotificationConsumer) Start() error {
	queues := []string{
		"notification.new_product",
		"notification.product_unpublished",
		"notification.auction_ended",
		"notification.auction_starting_ready",
	}

	for _, queue := range queues {
		go c.consume(queue)
	}

	// 启动死信队列消费者
	go c.consumeDLQ()

	return nil
}

// consume 消费消息
func (c *NotificationConsumer) consume(queueName string) {
	msgs, err := c.channel.Consume(
		queueName,
		"",     // consumer tag
		false,  // auto-ack (false = 手动确认)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Failed to register consumer for %s: %v", queueName, err)
		return
	}

	for msg := range msgs {
		var notification NotificationMessage
		if err := json.Unmarshal(msg.Body, &notification); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			msg.Ack(false) // 格式错误的消息直接确认，不重试
			continue
		}

		log.Printf("Processing notification: Type=%s, LiveStreamID=%d, ProductID=%d",
			notification.Type, notification.LiveStreamID, notification.ProductID)

		// 处理通知推送
		if err := c.handler.Handle(&notification); err != nil {
			log.Printf("Failed to process notification: %v", err)
			// 消费失败，进入死信队列
			msg.Nack(false, false)
		} else {
			// 成功确认
			msg.Ack(false)
		}
	}
}

// consumeDLQ 消费死信队列
func (c *NotificationConsumer) consumeDLQ() {
	msgs, err := c.channel.Consume(
		"notification_failed",
		"",     // consumer tag
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Printf("Failed to register DLQ consumer: %v", err)
		return
	}

	for msg := range msgs {
		var notification NotificationMessage
		if err := json.Unmarshal(msg.Body, &notification); err != nil {
			log.Printf("Failed to unmarshal DLQ message: %v", err)
			msg.Ack(false)
			continue
		}

		retryCount := 0
		if val, ok := msg.Headers["retry_count"].(int32); ok {
			retryCount = int(val)
		}

		if retryCount >= 3 {
			log.Printf("Max retries exceeded for notification: %+v", notification)
			// 记录到错误日志表或发送告警
			msg.Ack(false)
			continue
		}

		// 重试：根据消息类型选择合适的队列
		var targetQueue string
		if notification.Type == "auction_starting" {
			targetQueue = "notification.auction_starting_delayed"
		} else {
			targetQueue = "notification." + notification.Type
		}

		notification.RetryCount = retryCount + 1
		body, _ := json.Marshal(notification)

		err := c.channel.PublishWithContext(
			context.Background(),
			"notification.main",
			targetQueue,
			false, false,
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent,
				Headers: amqp.Table{
					"retry_count": retryCount + 1,
				},
			},
		)

		if err != nil {
			log.Printf("Failed to retry notification: %v", err)
			msg.Nack(false, true) // 重新入队
		} else {
			log.Printf("Retrying notification (attempt %d): %+v", retryCount+1, notification)
			msg.Ack(false)
		}
	}
}
