package measurement

import (
	"k8s.io/klog"
	"k8s.io/perf-tests/clusterloader2/pkg/measurement"
)

const (
	measurementName = "MyFancyMeasurement"
)

func SayHello() {

}

type customMeasurement struct{}

func metricConstructor() measurement.Measurement {
	return &customMeasurement{}
}

func init() {
	klog.V(2).Infof("Registering %s measurement", measurementName)
	if err := measurement.Register(measurementName, metricConstructor); err != nil {
		klog.Fatalf("Cannot register %s: %v", measurementName, err)
	}
}

func (*customMeasurement) Dispose() {}

func (*customMeasurement) Execute(config *measurement.Config) ([]measurement.Summary, error) {
	klog.V(0).Info("I am doing something fancy here")
	return nil, nil
}

func (*customMeasurement) String() string {
	return measurementName
}
