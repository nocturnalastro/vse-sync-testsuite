package collectors

type CollectedData interface{}

type Collector interface {
	Start() error       // Links collector to monitoring stack if required
	Get() CollectedData // Returns an interface to retreive data from the monitoring stack
	CleanUp() error     // Unlinks collecter from montioring stack if required
}
