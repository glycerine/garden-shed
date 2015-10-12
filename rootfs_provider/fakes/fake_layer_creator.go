// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
	"github.com/cloudfoundry-incubator/garden-shed/rootfs_provider"
)

type FakeLayerCreator struct {
	CreateStub        func(id string, parentImage *repository_fetcher.Image, shouldNamespace bool, quota int64) (string, []string, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		id              string
		parentImage     *repository_fetcher.Image
		shouldNamespace bool
		quota           int64
	}
	createReturns struct {
		result1 string
		result2 []string
		result3 error
	}
}

func (fake *FakeLayerCreator) Create(id string, parentImage *repository_fetcher.Image, shouldNamespace bool, quota int64) (string, []string, error) {
	fake.createMutex.Lock()
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		id              string
		parentImage     *repository_fetcher.Image
		shouldNamespace bool
		quota           int64
	}{id, parentImage, shouldNamespace, quota})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(id, parentImage, shouldNamespace, quota)
	} else {
		return fake.createReturns.result1, fake.createReturns.result2, fake.createReturns.result3
	}
}

func (fake *FakeLayerCreator) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeLayerCreator) CreateArgsForCall(i int) (string, *repository_fetcher.Image, bool, int64) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].id, fake.createArgsForCall[i].parentImage, fake.createArgsForCall[i].shouldNamespace, fake.createArgsForCall[i].quota
}

func (fake *FakeLayerCreator) CreateReturns(result1 string, result2 []string, result3 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 string
		result2 []string
		result3 error
	}{result1, result2, result3}
}

var _ rootfs_provider.LayerCreator = new(FakeLayerCreator)
