package queue

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDeliveryAttempt(t *testing.T) {
	cases := []struct {
		name        string
		redelivered bool
		headers     amqp.Table
		wantAttempt int
		wantKnown   bool
	}{
		{"本方计数头 x-retry-count=3", true, amqp.Table{retryCountHeader: int64(3)}, 3, true},
		{"x-retry-count 优先于 broker 头", true, amqp.Table{
			retryCountHeader:   int64(2),
			"x-delivery-count": int64(7),
		}, 2, true},
		{"首次投递无 header", false, nil, 1, true},
		{"x-delivery-count=0（首次）", false, amqp.Table{"x-delivery-count": int64(0)}, 1, true},
		{"x-delivery-count=1（第 2 次）", true, amqp.Table{"x-delivery-count": int64(1)}, 2, true},
		{"x-delivery-count int 宽度兼容", true, amqp.Table{"x-delivery-count": int(2)}, 3, true},
		{"x-death count=1", true, amqp.Table{"x-death": []any{amqp.Table{"count": int64(1)}}}, 2, true},
		{"x-death 多条目取最大", true, amqp.Table{"x-death": []any{
			amqp.Table{"count": int64(1)}, amqp.Table{"count": int64(3)},
		}}, 4, true},
		{"x-delivery-count 优先于 x-death", true, amqp.Table{
			"x-delivery-count": int64(1),
			"x-death":          []any{amqp.Table{"count": int64(9)}},
		}, 2, true},
		{"redelivered 但无任何计数 header", true, amqp.Table{"foo": "bar"}, 0, false},
		{"x-death 结构异常视为未知", true, amqp.Table{"x-death": "not-a-list"}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, known := deliveryAttempt(amqp.Delivery{Redelivered: tc.redelivered, Headers: tc.headers})
			if got != tc.wantAttempt || known != tc.wantKnown {
				t.Fatalf("got (%d,%v) want (%d,%v)", got, known, tc.wantAttempt, tc.wantKnown)
			}
		})
	}
}
