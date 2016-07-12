// This file was generated by counterfeiter
package repository_fetcherfakes

import (
	"net/url"
	"sync"

	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/cloudfoundry-incubator/garden-shed/repository_fetcher"
)

type FakeRepositoryFetcher struct {
	FetchStub        func(u *url.URL, diskQuota int64) (*repository_fetcher.Image, error)
	fetchMutex       sync.RWMutex
	fetchArgsForCall []struct {
		u         *url.URL
		diskQuota int64
	}
	fetchReturns struct {
		result1 *repository_fetcher.Image
		result2 error
	}
	FetchIDStub        func(u *url.URL) (layercake.ID, error)
	fetchIDMutex       sync.RWMutex
	fetchIDArgsForCall []struct {
		u *url.URL
	}
	fetchIDReturns struct {
		result1 layercake.ID
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRepositoryFetcher) Fetch(u *url.URL, diskQuota int64) (*repository_fetcher.Image, error) {
	fake.fetchMutex.Lock()
	fake.fetchArgsForCall = append(fake.fetchArgsForCall, struct {
		u         *url.URL
		diskQuota int64
	}{u, diskQuota})
	fake.recordInvocation("Fetch", []interface{}{u, diskQuota})
	fake.fetchMutex.Unlock()
	if fake.FetchStub != nil {
		return fake.FetchStub(u, diskQuota)
	} else {
		return fake.fetchReturns.result1, fake.fetchReturns.result2
	}
}

func (fake *FakeRepositoryFetcher) FetchCallCount() int {
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	return len(fake.fetchArgsForCall)
}

func (fake *FakeRepositoryFetcher) FetchArgsForCall(i int) (*url.URL, int64) {
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	return fake.fetchArgsForCall[i].u, fake.fetchArgsForCall[i].diskQuota
}

func (fake *FakeRepositoryFetcher) FetchReturns(result1 *repository_fetcher.Image, result2 error) {
	fake.FetchStub = nil
	fake.fetchReturns = struct {
		result1 *repository_fetcher.Image
		result2 error
	}{result1, result2}
}

func (fake *FakeRepositoryFetcher) FetchID(u *url.URL) (layercake.ID, error) {
	fake.fetchIDMutex.Lock()
	fake.fetchIDArgsForCall = append(fake.fetchIDArgsForCall, struct {
		u *url.URL
	}{u})
	fake.recordInvocation("FetchID", []interface{}{u})
	fake.fetchIDMutex.Unlock()
	if fake.FetchIDStub != nil {
		return fake.FetchIDStub(u)
	} else {
		return fake.fetchIDReturns.result1, fake.fetchIDReturns.result2
	}
}

func (fake *FakeRepositoryFetcher) FetchIDCallCount() int {
	fake.fetchIDMutex.RLock()
	defer fake.fetchIDMutex.RUnlock()
	return len(fake.fetchIDArgsForCall)
}

func (fake *FakeRepositoryFetcher) FetchIDArgsForCall(i int) *url.URL {
	fake.fetchIDMutex.RLock()
	defer fake.fetchIDMutex.RUnlock()
	return fake.fetchIDArgsForCall[i].u
}

func (fake *FakeRepositoryFetcher) FetchIDReturns(result1 layercake.ID, result2 error) {
	fake.FetchIDStub = nil
	fake.fetchIDReturns = struct {
		result1 layercake.ID
		result2 error
	}{result1, result2}
}

func (fake *FakeRepositoryFetcher) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.fetchMutex.RLock()
	defer fake.fetchMutex.RUnlock()
	fake.fetchIDMutex.RLock()
	defer fake.fetchIDMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeRepositoryFetcher) recordInvocation(key string, args []interface{}) {
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

var _ repository_fetcher.RepositoryFetcher = new(FakeRepositoryFetcher)