package repository_fetcher

import (
	"net/url"

	"code.cloudfoundry.org/garden-shed/layercake"
	"code.cloudfoundry.org/lager"
)

const MAX_ATTEMPTS = 3

type Retryable struct {
	RepositoryFetcher interface {
		Fetch(lager.Logger, *url.URL, int64) (*Image, error)
		FetchID(lager.Logger, *url.URL) (layercake.ID, error)
	}
}

func (retryable Retryable) Fetch(log lager.Logger, repoName *url.URL, diskQuota int64) (*Image, error) {
	var err error
	var response *Image
	for attempt := 1; attempt <= MAX_ATTEMPTS; attempt++ {
		response, err = retryable.RepositoryFetcher.Fetch(log, repoName, diskQuota)
		if err == nil {
			break
		}

		log.Error("failed-to-fetch", err, lager.Data{
			"attempt": attempt,
			"of":      MAX_ATTEMPTS,
		})
	}

	return response, err
}

func (retryable Retryable) FetchID(log lager.Logger, repoURL *url.URL) (layercake.ID, error) {
	var err error
	var response layercake.ID
	for attempt := 1; attempt <= MAX_ATTEMPTS; attempt++ {
		response, err = retryable.RepositoryFetcher.FetchID(log, repoURL)
		if err == nil {
			break
		}

		log.Error("failed-to-fetch-ID", err, lager.Data{
			"attempt": attempt,
			"of":      MAX_ATTEMPTS,
		})
	}

	return response, err
}
