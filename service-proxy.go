package service_proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ServiceApi struct {
	Method string
	Path   string
}

type HTTPServiceProxy struct {
	url          *url.URL
	preprocessor func(*http.Request) // 请求预处理器，用于统一添加请求头等操作
	apis         map[string]ServiceApi
}

type HTTPServiceProxyOptions struct {
	Scheme       string
	Host         string
	Preprocessor func(*http.Request)
	APIs         map[string]ServiceApi
}

func NewHTTPServiceProxy(opts HTTPServiceProxyOptions) *HTTPServiceProxy {
	apis := make(map[string]ServiceApi)

	if opts.APIs != nil {
		for key, api := range opts.APIs {
			apis[key] = api
		}
	}

	return &HTTPServiceProxy{
		url: &url.URL{
			Scheme: opts.Scheme,
			Host:   opts.Host,
		},
		preprocessor: opts.Preprocessor,
		apis:         apis,
	}
}

func (p *HTTPServiceProxy) getUrl(path string, query map[string]string) *url.URL {
	u := &url.URL{
		Scheme: p.url.Scheme,
		Host:   p.url.Host,
		Path:   path,
	}
	if query != nil {
		q := u.Query()
		for key, val := range query {
			q.Set(key, val)
		}
		u.RawQuery = q.Encode()
	}

	return u
}

func (p *HTTPServiceProxy) getUrlStr(path string, query map[string]string) string {
	return p.getUrl(path, query).String()
}

func (p *HTTPServiceProxy) RawRequest(req *http.Request) (result []byte, err error) {
	var (
		res *http.Response
	)
	client := &http.Client{}

	if p.preprocessor != nil {
		p.preprocessor(req)
	}

	res, err = client.Do(req)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to request (%s): %v", req.URL.String(), err))
		return
	}

	if res.StatusCode/100 != 2 {
		err = errors.New(fmt.Sprintf("Invalid status code of response: %d", res.StatusCode))
		return
	}

	result, err = ioutil.ReadAll(res.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to read response body from %s: %v", req.URL.String(), err))
		return
	}
	defer func() {
		_ = res.Body.Close()
	}()

	return
}

func (p *HTTPServiceProxy) getApi(key string) *ServiceApi {
	if p.apis != nil {
		if _, ok := p.apis[key]; ok {
			api := p.apis[key]
			return &api
		}
		return nil
	}
	return nil
}

func (p *HTTPServiceProxy) Request(apiKey string, body []byte, headers map[string]string) (result []byte, err error) {
	api := p.getApi(apiKey)
	if api == nil {
		return nil, errors.New(fmt.Sprintf("Invalid API key: %s", apiKey))
	}

	req, err := http.NewRequest(api.Method, p.getUrlStr(api.Path, nil), bytes.NewReader(body))

	header := make(http.Header)
	for key, val := range headers {
		header.Set(key, val)
	}

	return p.RawRequest(req)
}

func (p *HTTPServiceProxy) JSON(apiKey string, body []byte, headers map[string]string, result interface{}) error {
	data, err := p.Request(apiKey, body, headers)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}
