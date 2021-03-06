// This file was generated by counterfeiter
package rootfs_providerfakes

import (
	"sync"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/layercake"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
)

type FakeMetricser struct {
	MetricsStub        func(logger lager.Logger, id layercake.ID) (garden.ContainerDiskStat, error)
	metricsMutex       sync.RWMutex
	metricsArgsForCall []struct {
		logger lager.Logger
		id     layercake.ID
	}
	metricsReturns struct {
		result1 garden.ContainerDiskStat
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeMetricser) Metrics(logger lager.Logger, id layercake.ID) (garden.ContainerDiskStat, error) {
	fake.metricsMutex.Lock()
	fake.metricsArgsForCall = append(fake.metricsArgsForCall, struct {
		logger lager.Logger
		id     layercake.ID
	}{logger, id})
	fake.recordInvocation("Metrics", []interface{}{logger, id})
	fake.metricsMutex.Unlock()
	if fake.MetricsStub != nil {
		return fake.MetricsStub(logger, id)
	}
	return fake.metricsReturns.result1, fake.metricsReturns.result2
}

func (fake *FakeMetricser) MetricsCallCount() int {
	fake.metricsMutex.RLock()
	defer fake.metricsMutex.RUnlock()
	return len(fake.metricsArgsForCall)
}

func (fake *FakeMetricser) MetricsArgsForCall(i int) (lager.Logger, layercake.ID) {
	fake.metricsMutex.RLock()
	defer fake.metricsMutex.RUnlock()
	return fake.metricsArgsForCall[i].logger, fake.metricsArgsForCall[i].id
}

func (fake *FakeMetricser) MetricsReturns(result1 garden.ContainerDiskStat, result2 error) {
	fake.MetricsStub = nil
	fake.metricsReturns = struct {
		result1 garden.ContainerDiskStat
		result2 error
	}{result1, result2}
}

func (fake *FakeMetricser) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.metricsMutex.RLock()
	defer fake.metricsMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeMetricser) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ rootfs_provider.Metricser = new(FakeMetricser)
