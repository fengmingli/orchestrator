package retry

import (
	"time"
)

type Config struct {
	Times int
	Delay time.Duration
}

func Do(cfg Config, fn func() error) error {
	var err error
	for i := 0; i <= cfg.Times; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < cfg.Times {
			time.Sleep(cfg.Delay)
		}
	}
	return err
}
