package chatgpt

import (
	"sync"
	"time"
)

func NewDebouncer(interval time.Duration) func(f func()) {
	var l sync.Mutex
	var timer *time.Timer

	return func(f func()) {
		l.Lock()
		defer l.Unlock()
		// 使用lock保证d.timer更新之前一定先Stop.

		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(interval, f)
	}
}
