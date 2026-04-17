package observability

type Counter struct {
	name  string
	value int64
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Inc() {
	if c == nil {
		return
	}
	c.value++
}

func (c *Counter) Value() int64 {
	if c == nil {
		return 0
	}
	return c.value
}
