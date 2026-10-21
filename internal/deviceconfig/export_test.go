package deviceconfig

import "time"

// SetReloadPace shortens the wait for a reloaded device in tests.
func (p *Pushes) SetReloadPace(window, poll time.Duration) {
	p.reloadWindow, p.reloadPoll = window, poll
}
