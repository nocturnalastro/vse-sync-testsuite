package collectors

type CollectedData interface{}

type Collector interface {
	// interfaceName   string
	// ctx             clients.ContainerContext
	// DataTypes       [3]string
	// data            map[string]interface{}
	// inversePollRate float64
	// callback        Callback

	// running  map[string]bool
	// lastPoll time.Time

	Start(key string) error // Links collector to monitoring stack if required
	// Get() (CollectedData, error) // Returns an interface to retreive data from the monitoring stack
	ShouldPoll() bool           // Check if poll time has alapsed and if it should be polled again
	Poll() error                // Poll for collectables
	fetchLine() ([]byte, error) // Should call into callback
	CleanUp(key string) error   // Unlinks collecter from montioring stack if required
}
