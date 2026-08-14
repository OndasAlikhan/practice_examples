package circuit_breaker

type State string

const closed State = "closed"
const open State = "open"
const halfOpen State = "halfOpen"

func (s State) IsClosed() bool {
	return s == closed
}
func (s State) IsOpen() bool {
	return s == open
}
func (s State) IsHalfOpen() bool {
	return s == halfOpen
}

type Result string

const ok Result = "ok"
const fail Result = "fail"
