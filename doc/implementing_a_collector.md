# Implementing Collectors

Any collector must conform to the collector interface (TODO: link to collector interface). It should use the call back to expose is infomation to the user.
Once you have filled out your collector. Any arguments should be added to the `CollectionConstuctor` and method should also be defined on the `CollectionConstuctor`.
You will then need to add a call to the new method in the `setupCollectors` function in the runner package.
Finally you will need add your collector to `collectorNames` so that it gets started by the runner.

An example of a very simple collector:

In `collectors/collectors.go` any arguments additional should be added to the `CollectionConstuctor`
```go
...

type CollectionConstuctor struct {
    ...
    Msg string
}

...
```

In `collectors/anouncement_collector.go` you should define your collector and a constuctor method on `CollectionConstuctor`
```go
package collectors

import (
	"fmt"

	"github.com/redhat-partner-solutions/vse-sync-testsuite/pkg/callbacks"
)

type AnouncementCollector struct {
	callback callbacks.Callback
	key      string
	msg      string
}

func (anouncer *AnouncementCollector) Start(key string) error {
	anouncer.key = key
	return nil
}

func (anouncer *AnouncementCollector) ShouldPoll() bool {
	// We always want to annouce ourselves
	return true
}

func (anouncer *AnouncementCollector) Poll() []error {
	errs := make([]error, 1)
	err := anouncer.callback.Call(
		fmt.Sprintf("%T", anouncer),
		anouncer.key,
		anouncer.msg,
	)
	if err != nil {
		errs[0] = err
	}
	return errs
}

func (anouncer *AnouncementCollector) CleanUp(key string) error {
	return nil
}

func (constructor *CollectionConstuctor) NewAnouncementCollector() (*AnouncementCollector, error) {
	return &AnouncementCollector{msg: constructor.Msg}, nil
}


```
In runner/runner.go Call the `NewAnouncementCollector` constuctor in the `setupCollectors` function and append `"Anouncer"` to `collectorNames`
```go


func setupCollectors(
	collectorNames []string,
	callback callbacks.Callback,
	ptpInterface string,
	clientset *clients.Clientset,
	pollRate float64,
) []*collectors.Collector {
	collecterInstances := make([]*collectors.Collector, 0)
	var newCollector collectors.Collector

	constuctor := collectors.CollectionConstuctor{
		Callback:     callback,
		PTPInterface: ptpInterface,
		Clientset:    clientset,
		PollRate:     pollRate,
	}

	for _, constuctorName := range collectorNames {
		switch constuctorName {
		case "Anouncer":
            NewAnouncerCollector, err := constuctor.NewAnouncementCollector()
			// Handle error ...
			newCollector = NewAnouncerCollector
        ...
        }
        ...
    }
    ...
}
func Run(
	kubeConfig string,
	logLevel string,
	outputFile string,
	pollCount int,
	pollRate float64,
	ptpInterface string,
) {
	...
	collectorNames := make([]string, 0)
	collectorNames = append(collectorNames, "PTP", "Anouncer")
    ...
}
```
