package vault

import "time"

type Timer interface {
	Stop() bool
	Reset(d time.Duration) bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) AfterFunc(d time.Duration, f func()) Timer {
	return time.AfterFunc(d, f)
}

type Option func(*Vault)

func WithClock(c Clock) Option {
	return func(v *Vault) { v.clock = c }
}

func WithIdleTimeout(d time.Duration) Option {
	return func(v *Vault) { v.idleTimeout = d }
}

func WithOnLock(fn func()) Option {
	return func(v *Vault) { v.onLock = fn }
}
