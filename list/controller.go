package list

// Controller wraps a List in a channel-based interface.
type Controller struct {
	// list is the internal list managed by the Controller.
	list *List

	// TODO(CaptainHayashi): channels.
}

// NewController constructs a new Controller for a given List.
func NewController(l *List) *Controller {
	return &Controller{
		list: l,
	}
}

// Run runs this Controller's event loop.
func (c *Controller) Run() {
	// TODO(CaptainHayashi): actually run this
}
