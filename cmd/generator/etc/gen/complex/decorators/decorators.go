package decorators

import "github.com/test/complex/providers"

// @decorator named="app.service" priority=100
// MetricsDecorator adds metrics to the app service
func AddMetrics(
	service *providers.AppService,
	metrics MetricsCollector, // @inject named="metrics" optional=true
) *providers.AppService {
	return service
}

type MetricsCollector interface{}
