package nrelay

import (
	"strconv"

	"github.com/lafikl/consistent"
	"github.com/nats-io/nats.go"
	"github.com/octu0/chanque"
)

type distribute struct {
	mapping    map[string]chanque.Worker
	consistent *consistent.Consistent
	fallback   chanque.Worker
}

func (d *distribute) Publish(key string, msg *nats.Msg) bool {
	id, err := d.consistent.Get(key)
	if err != nil {
		return d.fallback.Enqueue(msg)
	}

	worker, ok := d.mapping[id]
	if ok != true {
		return d.fallback.Enqueue(msg)
	}
	return worker.Enqueue(msg)
}

func newDistribute(workers []chanque.Worker) *distribute {
	c := consistent.New()
	m := make(map[string]chanque.Worker, len(workers))
	for i, worker := range workers {
		id := strconv.Itoa(i)
		m[id] = worker
		c.Add(id)
	}
	return &distribute{
		mapping:    m,
		consistent: c,
		fallback:   workers[0],
	}
}
