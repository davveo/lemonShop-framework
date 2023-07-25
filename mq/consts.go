package mq

const (
	DefaultMaxConnection     = 5  //rabbitmq tcp 最大连接数
	DefaultMaxConsumeChannel = 25 //最大消费channel数(一般指消费者)
	DefaultMaxConsumeRetry   = 5  //消费者断线重连最大次数
	DefaultPushMaxTime       = 5  //最大重发次数
	DefaultMaxProductRetry   = 5  //生产者断线重连最大次数
	LoadBalanceRound         = 1  //轮循-连接池负载算法
)

const (
	RabbitmqTypePublish       = 1     //生产者
	RabbitmqTypeConsume       = 2     //消费者
	DefaultRetryMinRandomTime = 5000  //最小重试时间机数
	DefaultRetryMaxRandomTime = 15000 //最大重试时间机数

)

const (
	ExchangeTypeFanout = "fanout" //  Fanout：广播，将消息交给所有绑定到交换机的队列
	ExchangeTypeDirect = "direct" //Direct：定向，把消息交给符合指定routing key 的队列
	ExchangeTypeTopic  = "topic"  //Topic：通配符，把消息交给符合routing pattern（路由模式） 的队列
)

/*
错误码
*/
const (
	RcodePushMaxError                  = 501 //发送超过最大重试次数
	RcodeGetChannelError               = 502 //获取信道失败
	RcodeChannelQueueExchangeBindError = 503 //交换机/队列/绑定失败
	RcodeConnectionError               = 504 //连接失败
	RcodePushError                     = 505 //消息推送失败
	RcodeChannelCreateError            = 506 //信道创建失败
	RcodeRetryMaxError                 = 507 //超过最大重试次数

)
