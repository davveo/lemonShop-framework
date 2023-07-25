package mq

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RetryClientInterface interface {
	Push(pushData []byte) *RabbitMqError
	Ack() error
}

/*
*
重试工具
*/
type retryClient struct {
	channel          *amqp.Channel
	data             *amqp.Delivery
	header           map[string]interface{}
	deadExchangeName string
	deadQueueName    string
	deadRouteKey     string
	pool             *RabbitPool
	receive          *ConsumeReceive
}

func newRetryClient(
	channel *amqp.Channel,
	data *amqp.Delivery,
	header map[string]interface{},
	deadExchangeName string,
	deadQueueName string,
	deadRouteKey string,
	pool *RabbitPool,
	receive *ConsumeReceive) *retryClient {
	return &retryClient{
		channel:          channel,
		data:             data,
		header:           header,
		deadExchangeName: deadExchangeName,
		deadQueueName:    deadQueueName,
		deadRouteKey:     deadRouteKey,
		pool:             pool,
		receive:          receive}
}

func (r *retryClient) Ack() error {
	//如果是非自动确认消息 手动进行确认
	if !r.receive.IsAutoAck {

		if r.data != nil {
			return r.data.Ack(true)
		}
		return AckDataNil
	}
	return nil
}

func (r *retryClient) Push(pushData []byte) *RabbitMqError {
	if r.channel != nil {
		var retryNums int32
		retryNum, ok := r.header["retry_nums"]
		if !ok {
			retryNums = 0
		} else {
			retryNums = retryNum.(int32)
		}

		retryNums += 1

		if retryNums >= r.receive.MaxReTry {
			if r.receive.EventFail != nil {
				r.receive.EventFail(RcodeRetryMaxError, NewRabbitMqError(RcodeRetryMaxError, "The maximum number of retries exceeded. Procedure", ""), pushData)
			}
		} else {
			go func(tryNum int32, pushD []byte) {
				time.Sleep(time.Millisecond * 200)
				header := make(map[string]interface{}, 1)
				header["retry_nums"] = tryNum
				expirationTime, errs := RandomAround(r.pool.minRandomRetryTime, r.pool.maxRandomRetryTime)
				if errs != nil {
					expirationTime = 5000
				}

				err := r.channel.Publish(r.deadExchangeName, r.deadRouteKey, false, false, amqp.Publishing{
					ContentType:  "text/plain",
					Body:         pushD,
					Expiration:   strconv.FormatInt(expirationTime, 10),
					Headers:      r.header,
					DeliveryMode: amqp.Persistent,
				})
				if err != nil {
					if r.receive.EventFail != nil {
						r.receive.EventFail(RcodeRetryMaxError, NewRabbitMqError(RcodeRetryMaxError, "The maximum number of retries exceeded. Procedure", ""), pushD)
					}
				}

			}(retryNums, pushData)

		}
		return nil
	} else {
		return NewRabbitMqError(RcodeGetChannelError, fmt.Sprintf("获取队列 %s 的消费通道失败", r.deadQueueName), fmt.Sprintf("获取队列 %s 的消费通道失败", r.deadQueueName))
	}
}

// RabbitMqError 错误返回
type RabbitMqError struct {
	Code    int
	Message string
	Detail  string
}

func (e RabbitMqError) Error() string {
	return fmt.Sprintf("Exception (%d) Reason: %q", e.Code, e.Message)
}

func NewRabbitMqError(code int, message string, detail string) *RabbitMqError {
	return &RabbitMqError{Code: code, Message: message, Detail: detail}
}

type CallBack func(data []byte, header map[string]interface{}, retryClient RetryClientInterface) bool

// ConsumeReceive 消费者注册接收数据
type ConsumeReceive struct {
	ExchangeName string                   //交换机
	ExchangeType string                   //交换机类型
	Route        string                   //路由
	QueueName    string                   //队列名称
	EventSuccess CallBack                 //成功事件回调
	EventFail    func(int, error, []byte) //失败回调
	IsTry        bool                     //是否重试
	MaxReTry     int32                    //最大重式次数
	IsAutoAck    bool                     //是否自动确认
}

type RetryToolInterface interface {
	push()
}

type RetryTool struct {
	channel *amqp.Channel
}

func (r *RetryTool) push() {

}

/*
*
单个rabbitmq channel
*/
type rChannel struct {
	ch    *amqp.Channel
	index int32
}

type rConn struct {
	conn  *amqp.Connection
	index int32
}

type RabbitPool struct {
	minRandomRetryTime int64
	maxRandomRetryTime int64

	maxConnection int32 // 最大连接数量
	pushMaxTime   int   //最大重发次数

	connectionIndex   int32 //记录当前使用的连接
	connectionBalance int   //连接池负载算法

	channelPool map[int64]*rChannel //channel信道池
	connections map[int][]*rConn    // rabbitmq连接池

	channelLock    sync.RWMutex //信道池锁
	connectionLock sync.Mutex   //连接锁

	rabbitLoadBalance *RabbitLoadBalance //连接池负载模式(生产者)

	consumeMaxChannel   int32             //消费者最大信道数一般指消费者
	consumeReceive      []*ConsumeReceive //消费者注册事件
	consumeMaxRetry     int32             //消费者断线重连最大次数
	consumeCurrentRetry int32             //当前重连次数
	productMaxRetry     int32             //生产者重连次数
	productCurrentRetry int32
	pushCurrentRetry    int32 //当前推送重连交数

	clientType int //客户端类型 生产者或消费者 默认为生产者

	errorChanel chan *amqp.Error //错误捕捉channel

	connectStatus bool

	host        string //服务ip
	port        int    //服务端口
	user        string //用户名
	password    string //密码
	virtualHost string // 默认为/
}

// NewProductPool 初始化生产者
func NewProductPool() *RabbitPool {
	return newRabbitPool(RabbitmqTypePublish)
}

// NewConsumePool 初始化消费者
func NewConsumePool() *RabbitPool {
	return newRabbitPool(RabbitmqTypeConsume)
}

func newRabbitPool(clientType int) *RabbitPool {
	return &RabbitPool{
		minRandomRetryTime:  DefaultRetryMinRandomTime,
		maxRandomRetryTime:  DefaultRetryMaxRandomTime,
		clientType:          clientType,
		consumeMaxChannel:   DefaultMaxConsumeChannel,
		maxConnection:       DefaultMaxConnection,
		pushMaxTime:         DefaultPushMaxTime,
		connectionBalance:   LoadBalanceRound,
		connectionIndex:     0,
		consumeMaxRetry:     DefaultMaxConsumeRetry,
		consumeCurrentRetry: 0,
		productMaxRetry:     DefaultMaxProductRetry,
		pushCurrentRetry:    0,
		connectStatus:       false,
		connections:         make(map[int][]*rConn, 2),
		channelPool:         make(map[int64]*rChannel, 1),
		rabbitLoadBalance:   NewRabbitLoadBalance(),
		errorChanel:         make(chan *amqp.Error),
	}
}

// SetMaxConsumeChannel 设置消费者最大信道数
func (r *RabbitPool) SetMaxConsumeChannel(maxConsume int32) {
	r.consumeMaxChannel = maxConsume
}

/*
*
设置最大连接数
*/
func (r *RabbitPool) SetMaxConnection(maxConnection int32) {
	r.maxConnection = maxConnection
}

/*
*
设置随时重试时间
避免同一时刻一次重试过多
*/
func (r *RabbitPool) SetRandomRetryTime(min, max int64) {
	r.minRandomRetryTime = min
	r.maxRandomRetryTime = max
}

/*
*
设置连接池负载算法
默认轮循
*/
func (r *RabbitPool) SetConnectionBalance(balance int) {
	r.connectionBalance = balance
}

func (r *RabbitPool) GetHost() string {
	return r.host
}

func (r *RabbitPool) GetPort() int {
	return r.port
}

/*
*
连接rabbitmq
@param host string 服务器地址
@param port int 服务端口
@param user string 用户名
@param password 密码
*/
func (r *RabbitPool) Connect(host string, port int, user string, password string) error {
	r.host = host
	r.port = port
	r.user = user
	r.password = password
	r.virtualHost = "/"
	return r.initConnections(false)
}

/*
*
自定义虚拟机连接
@param host string 服务器地址
@param port int 服务端口
@param user string 用户名
@param password 密码
@param virtualHost虚拟机路径
*/
func (r *RabbitPool) ConnectVirtualHost(host string, port int, user string, password string, virtualHost string) error {
	r.host = host
	r.port = port
	r.user = user
	r.password = password
	r.virtualHost = virtualHost
	return r.initConnections(false)
}

/*
*
注册消费接收
*/
func (r *RabbitPool) RegisterConsumeReceive(consumeReceive *ConsumeReceive) {
	if consumeReceive != nil {
		r.consumeReceive = append(r.consumeReceive, consumeReceive)
	}
}

/*
*
消费者
*/
func (r *RabbitPool) RunConsume() error {
	r.clientType = RabbitmqTypeConsume
	if len(r.consumeReceive) == 0 {
		return errors.New("未注册消费者事件")
	}
	rConsume(r)
	return nil
}

/*
*
发送消息
*/
func (r *RabbitPool) Push(data *RabbitMqData) *RabbitMqError {
	return rPush(r, data, 1)
}

/*
*
获取当前连接
1.这里可以做负载算法, 默认使用轮循
*/
func (r *RabbitPool) getConnection() *rConn {
	changeConnectionIndex := r.connectionIndex
	currentIndex := r.rabbitLoadBalance.RoundRobin(changeConnectionIndex, r.maxConnection)
	currentNum := currentIndex - changeConnectionIndex
	atomic.AddInt32(&r.connectionIndex, currentNum)
	return r.connections[r.clientType][r.connectionIndex]
}

/*
*
获取信道
1.如果当前信道池不存在则创建
2.如果信息池存在则直接获取
3.每个连接池中连接维护一组信道
@param channelName string 信息道名称
*/
func (r *RabbitPool) getChannelQueue(conn *rConn, exChangeName string, exChangeType string, queueName string, route string, isDead bool, expireTime int) (*rChannel, error) {
	return r.getChannelQueueReset(conn, exChangeName, exChangeType, queueName, route, isDead, expireTime, false)
}

func (r *RabbitPool) deleteChannel(conn *rConn, exChangeName string, exChangeType string, queueName string, route string) {
	channelHashCode := channelHashCode(r.clientType, conn.index, exChangeName, exChangeType, queueName, route)
	if _, ok := r.channelPool[channelHashCode]; ok {
		delete(r.channelPool, channelHashCode)
	}
}

func (r *RabbitPool) getChannelQueueReset(
	conn *rConn,
	exChangeName string,
	exChangeType string,
	queueName string,
	route string,
	isDead bool,
	expireTime int,
	isReset bool) (*rChannel, error) {
	channelHashCode := channelHashCode(r.clientType, conn.index, exChangeName, exChangeType, queueName, route)
	if isReset {
		if _, ok := r.channelPool[channelHashCode]; ok {
			delete(r.channelPool, channelHashCode)
		}
	}
	if channelQueues, ok := r.channelPool[channelHashCode]; ok {
		return channelQueues, nil
	}
	//初始化channel
	//fmt.Printf("初始化channel")
	rChannel, err := r.initChannels(conn, exChangeName, exChangeType, queueName, route)
	if err != nil {
		return nil, err
	}
	channel, err := rDeclare(conn, r.clientType, rChannel, exChangeName, exChangeType, queueName, route, isDead, "", "", "")
	if err != nil {
		return nil, err
	}
	rChannel.ch = channel.ch
	r.channelPool[channelHashCode] = rChannel
	return rChannel, nil

}

/*
*
初始化连接池
*/
func (r *RabbitPool) initConnections(isLock bool) error {
	r.connections[r.clientType] = []*rConn{}
	var i int32 = 0
	for i = 0; i < r.maxConnection; i++ {
		itemConnection, err := rConnect(r, isLock)
		if err != nil {
			return err
		} else {
			r.connections[r.clientType] = append(r.connections[r.clientType], &rConn{conn: itemConnection, index: i})
		}
	}
	return nil
}

/*
*
初始化信道池
*/
func (r *RabbitPool) initChannels(
	conn *rConn,
	exChangeName string,
	exChangeType string,
	queueName string,
	route string) (*rChannel, error) {
	channel, err := rCreateChannel(conn)
	if err != nil {
		return nil, err
	}
	rChannel := &rChannel{ch: channel, index: 0}
	return rChannel, nil
}

/*
*
原rabbitmq连接
*/
func rConnect(r *RabbitPool, islock bool) (*amqp.Connection, error) {
	virtualHost := "/"
	if len(strings.TrimSpace(r.virtualHost)) > 0 {
		virtualHost = r.virtualHost
	}
	return connection(r.user, r.password, r.host, r.port, virtualHost)
}

func connection(user string, password string, host string, port int, vHost string) (*amqp.Connection, error) {
	virtualHost := "/"
	if len(strings.TrimSpace(vHost)) > 0 {
		virtualHost = vHost
	}
	connectionUrl := fmt.Sprintf("amqp://%s:%s@%s:%d%s", user, password, host, port, virtualHost)
	//fmt.Println(connectionUrl)
	client, err := amqp.Dial(connectionUrl)
	if err != nil {
		return nil, err
	}
	return client, nil
}

/*
*
创建rabbitmq信道
*/
func rCreateChannel(conn *rConn) (*amqp.Channel, error) {
	ch, err := conn.conn.Channel()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Create Connect Channel Error: %s", err.Error()))
	}
	return ch, nil
}

/*
*
绑定并声明
@param rconn *rConn tcp连接对象
@param clientType int 客户端类型
@param channel 信道
@param exChangeName 交换机名称
@param exChangeType 交换机类型
@param queueName 队列名称
@param route 路由key
@param isDeadQueue 是否是死信队列
@param deadQueueExpireTime int 死信队列到期时间
*/
func rDeclare(
	rconn *rConn,
	clientType int,
	channel *rChannel,
	exChangeName string,
	exChangeType string,
	queueName string,
	route string,
	isDeadQueue bool,
	oldExChangeName string,
	oldQueueName,
	oldRoute string) (*rChannel, error) {
	if clientType == RabbitmqTypePublish {
		if (len(exChangeType) == 0) || (exChangeType != ExchangeTypeDirect && exChangeType != ExchangeTypeFanout && exChangeType != ExchangeTypeTopic) {
			return channel, errors.New("交换机类型错误")
		}
	}
	newChannel := channel.ch
	err := newChannel.ExchangeDeclare(exChangeName, exChangeType,
		true, false, false, false, nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("MQ注册交换机失败:%s", err))
	}
	if (clientType != RabbitmqTypePublish && exChangeType != ExchangeTypeFanout) ||
		(clientType == RabbitmqTypeConsume && (exChangeType == ExchangeTypeFanout ||
			exChangeType == ExchangeTypeDirect)) {
		argsQue := make(map[string]interface{})
		if isDeadQueue {
			argsQue["x-dead-letter-exchange"] = oldExChangeName
			oldRoute = strings.TrimSpace(oldRoute)
			if len(oldRoute) > 0 {
				argsQue["x-dead-letter-routing-key"] = oldRoute
			}

		}
		queue, err := newChannel.QueueDeclare(queueName, true, false, false, false, argsQue)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("MQ注册队列失败:%s", err))
		}
		err = newChannel.QueueBind(queue.Name, route, exChangeName, false, nil)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("MQ绑定队列失败:%s", err))
		}
	}
	channel.ch = newChannel
	return channel, nil
}

/*
*
消费者处理
*/
func rConsume(pool *RabbitPool) {
	for _, v := range pool.consumeReceive {
		go func(pool *RabbitPool, receive *ConsumeReceive) {
			rListenerConsume(pool, receive)
		}(pool, v)
	}
	/**
	创建一个协程监听任务
	*/
	select {
	//case data := <-pool.errorChanel:
	case <-pool.errorChanel:
		statusLock.Lock()
		status = true
		statusLock.Unlock()
		retryConsume(pool)
	}

}

/*
*
重连处理
*/
func retryConsume(pool *RabbitPool) {
	log(fmt.Sprintf("2秒后开始重试:[%d]", pool.consumeCurrentRetry))
	atomic.AddInt32(&pool.consumeCurrentRetry, 1)
	time.Sleep(time.Second * 2)
	_, err := rConnect(pool, true)
	if err != nil {
		retryConsume(pool)
	} else {
		statusLock.Lock()
		status = false
		statusLock.Unlock()
		_ = pool.initConnections(false)
		rConsume(pool)
	}

}

/*
*
监听消费
*/
func rListenerConsume(pool *RabbitPool, receive *ConsumeReceive) {
	var i int32 = 0
	for i = 0; i < pool.consumeMaxChannel; i++ {
		itemI := i
		go func(num int32, p *RabbitPool, r *ConsumeReceive) {
			consumeTask(num, p, r)
		}(itemI, pool, receive)
	}
}

func setConnectError(pool *RabbitPool, code int, message string) {
	statusLock.Lock()
	defer statusLock.Unlock()

	if !status {
		pool.errorChanel <- &amqp.Error{
			Code:   code,
			Reason: message,
		}
	}
	status = true
}

// consumeTask 消费任务
func consumeTask(num int32, pool *RabbitPool, receive *ConsumeReceive) {
	//获取请求连接
	closeFlag := false
	pool.connectionLock.Lock()
	conn := pool.getConnection()
	pool.connectionLock.Unlock()
	//生成处理channel 根据最大channel数处理
	channel, err := rCreateChannel(conn)
	if err != nil {
		if receive.EventFail != nil {
			receive.EventFail(RcodeChannelCreateError,
				NewRabbitMqError(RcodeChannelCreateError, "channel create error", err.Error()), nil)
		}
		return
	}
	defer func() {
		_ = channel.Close()
		_ = conn.conn.Close()
	}()
	//defer
	notifyClose := make(chan *amqp.Error)
	closeChan := make(chan *amqp.Error, 1)
	rChanels := &rChannel{ch: channel, index: num}
	deadRChanels := &rChannel{ch: channel, index: num}

	deadExchangeName := fmt.Sprintf("%s-%s", receive.ExchangeName, "dead")
	deadQueueName := fmt.Sprintf("%s-%s", receive.QueueName, "dead")
	deadRouteKey := ""
	if len(receive.Route) > 0 {
		deadRouteKey = fmt.Sprintf("%s-%s", receive.Route, "dead")
	}

	//rChanels, err = rDeclare(conn, pool.clientType, rChanels, receive.ExchangeName,
	//receive.ExchangeType, receive.QueueName, receive.Route,
	//receive.IsDead, receive.DeadExchangeName, receive.DeadQueueName, receive.DeadRoute)
	rChanels, err = rDeclare(conn, pool.clientType, rChanels,
		receive.ExchangeName, receive.ExchangeType,
		receive.QueueName, receive.Route, false,
		"", "", "")
	//如果存在死信队列 则需要声明
	if receive.IsTry {
		if num%2 == 0 {
			deadChannel, deadErr := rCreateChannel(conn)
			if deadErr != nil {
				if receive.EventFail != nil {
					receive.EventFail(RcodeChannelCreateError,
						NewRabbitMqError(RcodeChannelCreateError, "dead channel create error", err.Error()), nil)
				}
				return
			}
			defer func() {
				_ = deadChannel.Close()
			}()

			deadRChanels, err = rDeclare(
				conn,
				pool.clientType,
				deadRChanels,
				deadExchangeName,
				ExchangeTypeDirect,
				deadQueueName,
				deadRouteKey,
				true,
				receive.ExchangeName,
				receive.QueueName,
				receive.Route)
		}
	}
	if err != nil {
		if receive.EventFail != nil {
			receive.EventFail(RcodeChannelQueueExchangeBindError,
				NewRabbitMqError(RcodeChannelQueueExchangeBindError, "交换机/队列/绑定失败", err.Error()), nil)
		}
		return
	}
	// 获取消费通道
	//确保rabbitmq会一个一个发消息
	_ = channel.Qos(1, 0, false)
	msgs, err := channel.Consume(
		receive.QueueName, // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if nil != err {
		if receive.EventFail != nil {
			receive.EventFail(RcodeGetChannelError, NewRabbitMqError(RcodeGetChannelError,
				fmt.Sprintf("获取队列 %s 的消费通道失败", receive.QueueName), err.Error()), nil)
		}
		return
	}

	//一旦消费者的channel有错误，产生一个amqp.Error，channel监听并捕捉到这个错误
	notifyClose = channel.NotifyClose(closeChan)
	for {
		select {
		case data := <-msgs:
			if receive.IsAutoAck { //如果是自动确认,否则需使用回调用 newRetryClient Ack
				_ = data.Ack(true)
			}
			if receive.EventSuccess != nil {
				retryClient := newRetryClient(channel, &data, data.Headers,
					deadExchangeName, deadQueueName, deadRouteKey, pool, receive)
				isOk := receive.EventSuccess(data.Body, data.Headers, retryClient)
				if !isOk && receive.IsTry {
					retryNum, ok := data.Headers["retry_nums"]
					var retryNums int32
					if !ok {
						retryNums = 0
					} else {
						retryNums = retryNum.(int32)
					}
					retryNums += 1
					if retryNums >= receive.MaxReTry {
						if receive.EventFail != nil {
							receive.EventFail(RcodeRetryMaxError, NewRabbitMqError(RcodeRetryMaxError,
								"The maximum number of retries exceeded. Procedure", ""), data.Body)
						}
					} else {
						go func(tryNum int32) {
							time.Sleep(time.Millisecond * 200)
							header := make(map[string]interface{}, 1)
							header["retry_nums"] = tryNum

							expirationTime, errs := RandomAround(pool.minRandomRetryTime, pool.maxRandomRetryTime)
							if errs != nil {
								expirationTime = 5000
							}

							//var reTryBody []byte
							//if len(reTryByte) == 0 {
							//	reTryBody = data.Body
							//} else {
							//	reTryBody = reTryByte
							//}
							err = channel.Publish(deadExchangeName, deadRouteKey, false, false, amqp.Publishing{
								ContentType:  "text/plain",
								Body:         data.Body,
								Expiration:   strconv.FormatInt(expirationTime, 10),
								Headers:      header,
								DeliveryMode: amqp.Persistent,
							})
						}(retryNums)
					}
				}
			}
		//一但有错误直接返回 并关闭信道
		case e := <-notifyClose:
			if receive.EventFail != nil {
				receive.EventFail(RcodeConnectionError, NewRabbitMqError(RcodeConnectionError, fmt.Sprintf("消息处理中断: queue:%s\n", receive.QueueName), e.Error()), nil)
			}
			setConnectError(pool, e.Code, fmt.Sprintf("消息处理中断: %s \n", e.Error()))
			closeFlag = true
		}
		if closeFlag {
			break
		}
	}
}

// tryConn 获取生产者连接
func tryConn(
	pool *RabbitPool,
	rc *rConn,
	currentTry int,
	isRetry bool,
	exchangeName string,
	exchangeType string,
	queueName string,
	route string) (*rConn, bool) {
	tryStatus := false
	if isRetry {
		tryStatus = true
		log("连接中断,2秒后开始重试")
		atomic.AddInt32(&pool.productCurrentRetry, 1)
		time.Sleep(time.Second * 2)
	}
	if rc == nil {
		rc = pool.getConnection()
	}
	if rc.conn == nil || rc.conn.IsClosed() {
		log("开始尝试重试连接")
		var err error
		pool.deleteChannel(rc, exchangeName, exchangeType, queueName, route)
		rc.conn, err = connection(pool.user, pool.password, pool.host, pool.port, pool.virtualHost)
		if err != nil {
			tryConn(pool, rc, currentTry, true, exchangeName, exchangeType, queueName, route)
		}
	}
	return rc, tryStatus
}

// rPush 发送消息
func rPush(pool *RabbitPool, data *RabbitMqData, sendTime int) *RabbitMqError {
	if sendTime >= pool.pushMaxTime {
		return NewRabbitMqError(RcodePushMaxError, "重试超过最大次数", "")
	}
	pool.channelLock.Lock()
	conn := pool.getConnection()
	conn, isTry := tryConn(pool, conn, 0,
		false, data.ExchangeName, data.ExchangeType, data.QueueName, data.Route)
	rChannels, err := pool.getChannelQueueReset(conn, data.ExchangeName,
		data.ExchangeType, data.QueueName, data.Route, false, 0, isTry)
	pool.channelLock.Unlock()
	if err != nil {
		return NewRabbitMqError(RcodeGetChannelError, "获取信道失败", err.Error())
	} else {
		err = rChannels.ch.Publish(data.ExchangeName, data.Route, false, false, amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(data.Data),
			DeliveryMode: amqp.Persistent, //持久化到磁盘
		})
		if err != nil { //如果消息发送失败, 重试发送
			//pool.channelLock.Unlock()
			//如果没有发送成功,休息两秒重发
			time.Sleep(time.Second * 2)
			sendTime++
			return rPush(pool, data, sendTime)
		}
	}
	return nil
}

// channelHashCode 信道hashcode
func channelHashCode(
	clientType int,
	connIndex int32,
	exChangeName string,
	exChangeType string,
	queueName string,
	route string) int64 {
	hashStr := fmt.Sprintf("%d-%d-%s-%s-%s-%s", clientType,
		connIndex, exChangeName, exChangeType, queueName, route)
	channelHashCode := hashCode(hashStr)
	return channelHashCode
}
