package mq

import (
	"fmt"
	"sync"
	"testing"
)

func TestConsume(t *testing.T) {
	initConsumerabbitmq()
	Consume()
}

func Consume() {
	nomrl := &ConsumeReceive{
		ExchangeName: "testChange32", //队列名称
		ExchangeType: ExchangeTypeDirect,
		Route:        "testRoute32",
		QueueName:    "testQueue32",
		IsTry:        true,  //是否重试
		IsAutoAck:    false, //自动消息确认
		MaxReTry:     5,     //最大重试次数
		EventFail: func(code int, e error, data []byte) {
			fmt.Printf("error:%s", e)
		},
		EventSuccess: func(data []byte,
			header map[string]interface{},
			retryClient RetryClientInterface) bool { //如果返回true 则无需重试
			_ = retryClient.Ack()
			fmt.Printf("data:%s\n", string(data))
			return true
		},
	}
	instanceConsumePool.RegisterConsumeReceive(nomrl)
	err := instanceConsumePool.RunConsume()
	if err != nil {
		fmt.Println(err)
	}
}

var onceConsumePool sync.Once
var instanceConsumePool *RabbitPool

func initConsumerabbitmq() *RabbitPool {
	onceConsumePool.Do(func() {
		instanceConsumePool = NewConsumePool()
		//instanceConsumePool.SetMaxConsumeChannel(100)
		//err := instanceConsumePool.Connect("192.168.1.169", 5672, "admin", "admin")
		err := instanceConsumePool.ConnectVirtualHost("192.168.186.130", 5672, "guest", "guest", "/temptest1")
		if err != nil {
			fmt.Println(err)
		}
	})
	return instanceConsumePool
}
