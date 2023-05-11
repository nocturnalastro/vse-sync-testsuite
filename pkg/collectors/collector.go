package collectors

type CollectedData interface{}

type Collector interface {
	// invese_poll_rate float64
	// callback         Callback
	// running          []string
	// last_poll        time.Time

	Start(key string) error // Links collector to monitoring stack if required
	// Get() (CollectedData, error) // Returns an interface to retreive data from the monitoring stack
	ShouldPoll() bool           // Check if poll time has alapsed and if it should be polled again
	Poll() error                // Poll for collectables
	fetchLine() ([]byte, error) // Should call into callback
	CleanUp(key string) error   // Unlinks collecter from montioring stack if required
}
