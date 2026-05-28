package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"errors"
	"time"
)

// RabbitMQConfig RabbitMQ配置
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

// RabbitMQConnection RabbitMQ连接管理
type RabbitMQConnection struct {
	config  *RabbitMQConfig
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQConnection 创建RabbitMQ连接
func NewRabbitMQConnection(config *RabbitMQConfig) (*RabbitMQConnection, error) {
	url := amqp.URI{
		Scheme:   "amqp",
		Host:     config.Host,
		Port:     5672,
		Username: config.User,
		Password: config.Password,
		Vhost:    config.VHost,
	}.String()

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	rmq := &RabbitMQConnection{
		config:  config,
		conn:    conn,
		channel: ch,
	}

	// 初始化交换机和队列
	if err := rmq.initExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, err
	}

	return rmq, nil
}

// initExchangesAndQueues 初始化交换机和队列（使用DLX + TTL实现延迟队列）
func (r *RabbitMQConnection) initExchangesAndQueues() error {
	// 1. 创建主交换机（direct）
	err := r.channel.ExchangeDeclare(
		"notification.main", // name
		"direct",            // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}

	// 2. 创建死信交换机
	err = r.channel.ExchangeDeclare(
		"notification.dlx", // name
		"direct",           // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return err
	}

	// 3. 创建立即队列（新商品发布、商品下架、竞拍结束）
	immediateQueues := []string{
		"notification.new_product",
		"notification.product_unpublished",
		"notification.auction_ended",
	}

	for _, queueName := range immediateQueues {
		_, err = r.channel.QueueDeclare(
			queueName,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			amqp.Table{
				"x-dead-letter-exchange":    "notification.dlx",
				"x-dead-letter-routing-key": "failed",
			},
		)
		if err != nil {
			return err
		}
	}

	// 4. 创建延迟队列（竞拍开始前30分钟通知）- 使用TTL + DLX实现
	// 延迟队列配置：消息过期后自动转发到主交换机
	_, err = r.channel.QueueDeclare(
		"notification.auction_starting_delayed", // 队列名
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "notification.main",      // 过期后转发到主交换机
			"x-dead-letter-routing-key": "auction_starting_ready", // 使用这个routing key
			"x-message-ttl":             1800000,                  // 默认TTL 30分钟（毫秒），可被消息级别的TTL覆盖
		},
	)
	if err != nil {
		return err
	}

	// 5. 创建竞拍开始就绪队列（从延迟队列转发过来的消息）
	_, err = r.channel.QueueDeclare(
		"notification.auction_starting_ready",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "notification.dlx",
			"x-dead-letter-routing-key": "failed",
		},
	)
	if err != nil {
		return err
	}

	// 6. 创建死信队列（失败重试）
	_, err = r.channel.QueueDeclare(
		"notification_failed",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	// 7. 绑定立即队列到主交换机
	routingBindings := map[string]string{
		"notification.new_product":        "new_product",
		"notification.product_unpublished": "product_unpublished",
		"notification.auction_ended":      "auction_ended",
		"notification.auction_starting_ready": "auction_starting_ready",
	}

	for queueName, routingKey := range routingBindings {
		err = r.channel.QueueBind(queueName, routingKey, "notification.main", false, nil)
		if err != nil {
			return err
		}
	}

	// 8. 绑定死信队列到死信交换机
	err = r.channel.QueueBind("notification_failed", "failed", "notification.dlx", false, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetChannel 获取Channel
func (r *RabbitMQConnection) GetChannel() *amqp.Channel {
	return r.channel
}

// Close 关闭连接
func (r *RabbitMQConnection) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Reconnect 重连机制
func (r *RabbitMQConnection) Reconnect() error {
	for i := 0; i < 5; i++ {
		time.Sleep(time.Duration(i+1) * time.Second)

		conn, err := amqp.Dial(r.config.Host)
		if err != nil {
			log.Printf("Reconnect attempt %d failed: %v", i+1, err)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			log.Printf("Channel creation attempt %d failed: %v", i+1, err)
			continue
		}

		r.conn = conn
		r.channel = ch

		// 重新初始化交换机和队列
		if err := r.initExchangesAndQueues(); err != nil {
			r.Close()
			log.Printf("Exchange/Queue init attempt %d failed: %v", i+1, err)
			continue
		}

		log.Println("Successfully reconnected to RabbitMQ")
		return nil
	}

	return errors.New("failed to reconnect after 5 attempts")
}
