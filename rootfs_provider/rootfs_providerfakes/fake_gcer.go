// This file was generated by counterfeiter
package rootfs_providerfakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
)

type FakeGCer struct {
	GCStub        func(log lager.Logger, cake layercake.Cake) error
	gCMutex       sync.RWMutex
	gCArgsForCall []struct {
		log  lager.Logger
		cake layercake.Cake
	}
	gCReturns struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeGCer) GC(log lager.Logger, cake layercake.Cake) error {
	fake.gCMutex.Lock()
	fake.gCArgsForCall = append(fake.gCArgsForCall, struct {
		log  lager.Logger
		cake layercake.Cake
	}{log, cake})
	fake.recordInvocation("GC", []interface{}{log, cake})
	fake.gCMutex.Unlock()
	if fake.GCStub != nil {
		return fake.GCStub(log, cake)
	} else {
		return fake.gCReturns.result1
	}
}

func (fake *FakeGCer) GCCallCount() int {
	fake.gCMutex.RLock()
	defer fake.gCMutex.RUnlock()
	return len(fake.gCArgsForCall)
}

func (fake *FakeGCer) GCArgsForCall(i int) (lager.Logger, layercake.Cake) {
	fake.gCMutex.RLock()
	defer fake.gCMutex.RUnlock()
	return fake.gCArgsForCall[i].log, fake.gCArgsForCall[i].cake
}

func (fake *FakeGCer) GCReturns(result1 error) {
	fake.GCStub = nil
	fake.gCReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeGCer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.gCMutex.RLock()
	defer fake.gCMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeGCer) recordInvocation(key string, args []interface{}) {
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

var _ rootfs_provider.GCer = new(FakeGCer)
