package repository_fetcher

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/docker/image"

	"github.com/cloudfoundry-incubator/garden-shed/distclient"
	"github.com/cloudfoundry-incubator/garden-shed/layercake"
	"github.com/pivotal-golang/lager"
)

type Remote struct {
	DefaultHost string
	Dial        Dialer
	Cake        layercake.Cake
	Logger      lager.Logger
}

func NewRemote(logger lager.Logger, defaultHost string, cake layercake.Cake, dialer Dialer) *Remote {
	return &Remote{
		DefaultHost: defaultHost,
		Dial:        dialer,
		Cake:        cake,
		Logger:      logger,
	}
}

func (r *Remote) manifest(u *url.URL) (distclient.Conn, *distclient.Manifest, error) {
	host := u.Host
	if host == "" {
		host = r.DefaultHost
	}

	path := u.Path[1:] // strip off initial '/'
	if strings.Index(path, "/") < 0 {
		path = "library/" + path
	}

	tag := u.Fragment
	if tag == "" {
		tag = "latest"
	}

	conn, err := r.Dial.Dial(r.Logger, host, path)
	if err != nil {
		return nil, nil, err
	}

	manifest, err := conn.GetManifest(r.Logger, tag)
	if err != nil {
		return nil, nil, fmt.Errorf("get manifest for tag %s on repo %s: %s", u.Fragment, u, err)
	}

	return conn, manifest, err
}

func (r *Remote) Fetch(u *url.URL, diskQuota int64) (*Image, error) {
	conn, manifest, err := r.manifest(u)
	if err != nil {
		return nil, err
	}

	for _, layer := range manifest.Layers {
		if _, err := r.Cake.Get(layercake.DockerImageID(layer.ID)); err == nil {
			continue // got cache
		}

		blob, err := conn.GetBlobReader(r.Logger, layer.Digest)
		if err != nil {
			return nil, err
		}

		if err := r.Cake.Register(&image.Image{ID: layer.ID, Parent: layer.Parent}, blob); err != nil {
			return nil, err
		}
	}

	return &Image{ImageID: manifest.Layers[len(manifest.Layers)-1].ID}, nil
}

func (r *Remote) FetchID(u *url.URL) (layercake.ID, error) {
	_, manifest, err := r.manifest(u)
	if err != nil {
		return nil, err
	}

	return layercake.DockerImageID(manifest.Layers[len(manifest.Layers)-1].ID), nil
}

//go:generate counterfeiter . Dialer
type Dialer interface {
	Dial(logger lager.Logger, host, repo string) (distclient.Conn, error)
}

type DialFunc func(logger lager.Logger, host, repo string) (distclient.Conn, error)

func (fn DialFunc) Dial(logger lager.Logger, host, repo string) (distclient.Conn, error) {
	return fn(logger, host, repo)
}
