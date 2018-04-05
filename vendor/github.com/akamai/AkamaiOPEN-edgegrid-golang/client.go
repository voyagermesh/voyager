// Package edgegrid provides the Akamai OPEN Edgegrid Authentication scheme
//
// Deprecated: use client-v1/client instead
package edgegrid

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strings"
)

const (
	libraryVersion = "0.3.0"
)

// Response is a Wrapper for http.Response
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
type Response http.Response

// Client is a simple http.Client wrapper that transparently signs requests
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
type Client struct {
	http.Client

	// HTTP client used to communicate with the Akamai APIs.
	//client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	Config Config
}

// JSONBody represents a generic JSON response
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
type JSONBody map[string]interface{}

// New creates a new Client with a given Config
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func New(httpClient *http.Client, config Config) (*Client, error) {
	c := NewClient(httpClient)
	c.Config = config

	baseURL, err := url.Parse("https://" + config.Host)

	if err != nil {
		return nil, err
	}

	c.BaseURL = baseURL
	return c, nil
}

// NewClient creates a new Client
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &Client{
		Client: *httpClient,
		UserAgent: "Akamai-Open-Edgegrid-golang/" + libraryVersion +
			" golang/" + strings.TrimPrefix(runtime.Version(), "go"),
	}

	return client
}

// NewRequest creates an API request. A relative URL can be provided in urlStr, which will be resolved to the
// BaseURL of the Client. If specified, the value pointed to by body is JSON encoded and included in as the request body.
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	var req *http.Request

	urlStr = strings.TrimPrefix(urlStr, "/")

	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	req, err = http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", c.UserAgent)

	return req, nil
}

// NewJSONRequest creates an API request with JSON body. A relative URL can be provided in urlStr, which will be resolved to the
// BaseURL of the Client. If specified, the value pointed to by body is JSON encoded and included in as the request body.
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) NewJSONRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(method, urlStr, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json,*/*")

	return req, nil
}

// Do executes the HTTP request
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) Do(req *http.Request) (*Response, error) {
	req = c.Config.AddRequestHeader(req)
	response, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	res := Response(*response)
	return &res, nil
}

// Get performs an HTTP GET request
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) Get(url string) (*Response, error) {
	req, err := c.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req = c.Config.AddRequestHeader(req)
	response, err := c.Do(req)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Post performs an HTTP POST request
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) Post(url string, bodyType string, body interface{}) (*Response, error) {
	req, err := c.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)

	req = c.Config.AddRequestHeader(req)
	response, err := c.Do(req)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// PostForm performs an HTTP POST request with a form body
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) PostForm(url string, data url.Values) (*Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// PostJSON performs an HTTP POST request with a JSON body
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) PostJSON(url string, data interface{}) (*Response, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return nil, err
	}

	return c.Post(url, "application/json", buf)
}

// Head performs an HTTP HEAD request
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (c *Client) Head(url string) (*Response, error) {
	req, err := c.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// BodyJSON retrieves and unmarshals a responses JSON body
//
// Deprecated: use github.com/akamai/AkamaiOPEN-edgegrid-golang/client
func (r *Response) BodyJSON(data interface{}) error {
	if data == nil {
		return errors.New("You must pass in an interface{}")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &data)

	return err
}
