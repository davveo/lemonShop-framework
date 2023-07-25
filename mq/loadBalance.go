package mq

// RabbitLoadBalance 连接负载处理
type RabbitLoadBalance struct {
}

func NewRabbitLoadBalance() *RabbitLoadBalance {
	return &RabbitLoadBalance{}
}

// RoundRobin 轮循
func (r *RabbitLoadBalance) RoundRobin(cIndex, max int32) int32 {
	if max == 0 {
		return 0
	}
	return (cIndex + 1) % max
}
