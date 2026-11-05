package queue

import (
	"errors"
	"fmt"
	"testing"

	"common/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewTaskTopologyDefaults(t *testing.T) {
	var cfg config.Config
	cfg.Queue.RabbitMQ.Queue = "my_go_study.tasks"
	topo := newTaskTopology(cfg)
	if topo.queue != "my_go_study.tasks" || topo.dlq != "my_go_study.tasks.dlq" || topo.dlx != "my_go_study.tasks.dlx" {
		t.Fatalf("topo=%+v", topo)
	}
	args := topo.queueArgs()
	if args["x-dead-letter-exchange"] != "my_go_study.tasks.dlx" {
		t.Fatalf("dlx arg=%v", args["x-dead-letter-exchange"])
	}
	// routing key 必须等于队列名：发布走默认 exchange，rk 就是队列名。
	if args["x-dead-letter-routing-key"] != "my_go_study.tasks" {
		t.Fatalf("rk arg=%v", args["x-dead-letter-routing-key"])
	}
}

func TestNewTaskTopologyOverrides(t *testing.T) {
	var cfg config.Config
	cfg.Queue.RabbitMQ.Queue = "q1"
	cfg.Queue.RabbitMQ.DLQ = "custom.dlq"
	cfg.Queue.RabbitMQ.DLX = "custom.dlx"
	topo := newTaskTopology(cfg)
	if topo.dlq != "custom.dlq" || topo.dlx != "custom.dlx" {
		t.Fatalf("topo=%+v", topo)
	}
}

func TestIsPreconditionFailed(t *testing.T) {
	// amqp091 可能以值或指针返回，两种都要认。
	if !isPreconditionFailed(amqp.Error{Code: amqp.PreconditionFailed}) {
		t.Error("值类型 406 未识别")
	}
	if !isPreconditionFailed(&amqp.Error{Code: amqp.PreconditionFailed}) {
		t.Error("指针类型 406 未识别")
	}
	if !isPreconditionFailed(fmt.Errorf("declare queue: %w", amqp.Error{Code: amqp.PreconditionFailed})) {
		t.Error("包装后的 406 未识别")
	}
	if isPreconditionFailed(&amqp.Error{Code: amqp.NotFound}) {
		t.Error("404 被误判为 406")
	}
	if isPreconditionFailed(errors.New("dial tcp: connection refused")) {
		t.Error("普通错误被误判为 406")
	}
	if isPreconditionFailed(nil) {
		t.Error("nil 被误判为 406")
	}
}
