package plugins

// Keptn implements the plugin interface
type Keptn struct{}

// Init initializes the plugin by interacting with the plugin components
func (p *Keptn) Init() error {
	panic("Implement me")
}

func (p *Keptn) Name() string {
	return "keptn"
}
