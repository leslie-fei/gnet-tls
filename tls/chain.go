package tls

import "errors"

type chain struct {
	calls []func() error
	err   error
	idx   int
}

func (c *chain) then(fn func() error) *chain {
	c.calls = append(c.calls, fn)
	return c
}

func (c *chain) exec() error {
	if c.err != nil {
		return c.err
	}
	if c.idx >= len(c.calls) {
		return errors.New("chain calls current idx >= calls length")
	}
	c.err = c.calls[c.idx]()
	c.idx++
	// is last clear
	if c.idx == len(c.calls) {
		c.calls = nil
	}
	return c.err
}
