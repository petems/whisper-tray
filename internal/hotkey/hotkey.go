package hotkey

// Manager defines the interface for global hotkey management
type Manager interface {
	Register(accel string, callback func(pressed bool)) error
	Unregister(accel string) error
	Close() error
}