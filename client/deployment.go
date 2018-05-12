package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/ubclaunchpad/inertia/common"
	git "gopkg.in/src-d/go-git.v4"
)

// Deployment manages a deployment
type Deployment struct {
	*RemoteVPS
	Repository *git.Repository
	Auth       string
	Project    string
	BuildType  string
}

// GetDeployment returns the local deployment setup
func GetDeployment(name string) (*Deployment, error) {
	config, err := GetProjectConfigFromDisk()
	if err != nil {
		return nil, err
	}

	repo, err := common.GetLocalRepo()
	if err != nil {
		return nil, err
	}

	remote, found := config.GetRemote(name)
	if !found {
		return nil, errors.New("Remote not found")
	}
	auth := remote.Daemon.Token

	return &Deployment{
		RemoteVPS:  remote,
		Repository: repo,
		Auth:       auth,
		BuildType:  config.BuildType,
		Project:    config.Project,
	}, nil
}

// Up brings the project up on the remote VPS instance specified
// in the deployment object.
func (d *Deployment) Up(buildType string, stream bool) (*http.Response, error) {
	// TODO: Support other Git remotes.
	origin, err := d.Repository.Remote("origin")
	if err != nil {
		return nil, err
	}

	if buildType == "" {
		buildType = d.BuildType
	}

	reqContent := &common.DaemonRequest{
		Stream:    stream,
		Project:   d.Project,
		BuildType: buildType,
		Secret:    d.RemoteVPS.Daemon.Secret,
		GitOptions: &common.GitOptions{
			RemoteURL: common.GetSSHRemoteURL(origin.Config().URLs[0]),
			Branch:    d.Branch,
		},
	}
	return d.post("/up", reqContent)
}

// Down brings the project down on the remote VPS instance specified
// in the configuration object.
func (d *Deployment) Down() (*http.Response, error) {
	return d.post("/down", nil)
}

// Status lists the currently active containers on the remote VPS instance
func (d *Deployment) Status() (*http.Response, error) {
	resp, err := d.get("/status", nil)
	if err != nil &&
		(strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "refused")) {
		return nil, fmt.Errorf("daemon on remote %s appears offline or inaccessible", d.Name)
	}
	return resp, err
}

// Reset shuts down deployment and deletes the contents of the deployment's
// project directory
func (d *Deployment) Reset() (*http.Response, error) {
	return d.post("/reset", nil)
}

// Logs get logs of given container
func (d *Deployment) Logs(stream bool, container string) (*http.Response, error) {
	reqContent := map[string]string{
		common.Stream:    strconv.FormatBool(stream),
		common.Container: container,
	}

	return d.get("/logs", reqContent)
}

// AddUser adds an authorized user for access to Inertia Web
func (d *Deployment) AddUser(username, password string, admin bool) (*http.Response, error) {
	reqContent := &common.UserRequest{
		Username: username,
		Password: password,
		Admin:    admin,
	}
	return d.post("/user/adduser", reqContent)
}

// RemoveUser prevents a user from accessing Inertia Web
func (d *Deployment) RemoveUser(username string) (*http.Response, error) {
	reqContent := &common.UserRequest{Username: username}
	return d.post("/user/removeuser", reqContent)
}

// ResetUsers resets all users on the remote.
func (d *Deployment) ResetUsers() (*http.Response, error) {
	return d.post("/user/resetusers", nil)
}

// ListUsers lists all users on the remote.
func (d *Deployment) ListUsers() (*http.Response, error) {
	return d.get("/user/listusers", nil)
}

// Sends a GET request. "queries" contains query string arguments.
func (d *Deployment) get(endpoint string, queries map[string]string) (*http.Response, error) {
	// Assemble request
	req, err := d.buildRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Add query strings
	if queries != nil {
		q := req.URL.Query()
		for k, v := range queries {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	client := buildHTTPSClient()
	return client.Do(req)
}

func (d *Deployment) post(endpoint string, requestBody interface{}) (*http.Response, error) {
	// Assemble payload
	var payload io.Reader
	if requestBody != nil {
		body, err := json.Marshal(requestBody)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(body)
	} else {
		payload = nil
	}

	// Assemble request
	req, err := d.buildRequest("POST", endpoint, payload)
	if err != nil {
		return nil, err
	}

	client := buildHTTPSClient()
	return client.Do(req)
}

func (d *Deployment) buildRequest(method string, endpoint string, payload io.Reader) (*http.Request, error) {
	// Assemble URL
	url, err := url.Parse("https://" + d.RemoteVPS.GetIPAndPort())
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, endpoint)
	urlString := url.String()

	// Assemble request
	req, err := http.NewRequest(method, urlString, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.Auth)

	return req, nil
}

func buildHTTPSClient() *http.Client {
	// Make HTTPS request
	tr := &http.Transport{
		// Our certificates are self-signed, so will raise
		// a warning - currently, we ask our client to ignore
		// this warning.
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return &http.Client{Transport: tr}
}
