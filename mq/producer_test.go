package mq

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestProduct(t *testing.T) {
	initrabbitmq()
	waitall()
}

var oncePool sync.Once
var instanceRPool *RabbitPool

func initrabbitmq() *RabbitPool {
	oncePool.Do(func() {
		instanceRPool = NewProductPool()
		//err := instanceRPool.Connect("192.168.1.169", 5672, "admin", "admin")
		err := instanceRPool.ConnectVirtualHost("192.168.186.130", 5672, "guest", "guest", "/temptest1")
		if err != nil {
			fmt.Println(err)
		}
	})
	return instanceRPool
}

func waitall() {
	var num int = 0
	for {
		num++
		data := GetRabbitMqDataFormat("testChange32", ExchangeTypeDirect, "testQueue32", "testRoute32", fmt.Sprintf("这里是数据%d", num))
		err := instanceRPool.Push(data)
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Second)
	}
}

func rund() {

	var wg sync.WaitGroup

	//wg.Add(1)
	//go func() {
	//	fmt.Println("aaaaaaaaaaaaaaaaaaaaaa")
	//	defer wg.Done()
	//	runtime.SetMutexProfileFraction(1)  // 开启对锁调用的跟踪
	//	runtime.SetBlockProfileRate(1)      // 开启对阻塞操作的跟踪
	//	err:= http.ListenAndServe("0.0.0.0:8080", nil)
	//	fmt.Println(err)
	//}()

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			data := GetRabbitMqDataFormat("testChange31", ExchangeTypeDirect, "testQueue31", "", fmt.Sprintf("这里是数据%d", num))
			err := instanceRPool.Push(data)
			if err != nil {
				fmt.Println(err)
			}
		}(i)
	}

	wg.Wait()
}
