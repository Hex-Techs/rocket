package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/models/module"
	"github.com/hex-techs/rocket/pkg/models/project"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

// Interface defines the interface for CRUD operations on clusters
type Interface interface {
	Create(obj interface{}) error
	Delete(id string, obj interface{}) error
	Update(id string, obj interface{}) error
	Get(id string, query *web.Request, obj interface{}) error
	List(query *web.Request, obj interface{}) error
}

func New(manager, token string) Interface {
	return &clientset{
		cli:     http.DefaultClient,
		manager: manager,
		token:   token,
	}
}

type clientset struct {
	cli     *http.Client
	manager string
	token   string
}

func (c *clientset) Create(obj interface{}) error {
	result, err := c.sendRequest("", http.MethodPost, nil, obj)
	if err != nil {
		return err
	}
	if result.State.Code != 0 {
		return errors.New(result.State.Msg)
	}
	return nil
}

func (c *clientset) Delete(id string, obj interface{}) error {
	result, err := c.sendRequest(id, http.MethodDelete, nil, obj)
	if err != nil {
		return err
	}
	if result.State.Code != 0 {
		return errors.New(result.State.Msg)
	}
	return nil
}

func (c *clientset) Update(id string, obj interface{}) error {
	result, err := c.sendRequest(id, http.MethodPut, nil, obj)
	if err != nil {
		return err
	}
	if result.State.Code != 0 {
		return errors.New(result.State.Msg)
	}
	return nil
}

func (c *clientset) Get(id string, query *web.Request, obj interface{}) error {
	result, err := c.sendRequest(id, http.MethodGet, query, obj)
	if err != nil {
		return err
	}
	if result.State.Code != 0 {
		return errors.New(result.State.Msg)
	}
	b, err := json.Marshal(result.Data)
	if err != nil {
		return fmt.Errorf("marshal data with error: %v", err)
	}
	if err := json.Unmarshal(b, obj); err != nil {
		return fmt.Errorf("unmarshal data with error: %v", err)
	}
	return nil
}

func (c *clientset) List(query *web.Request, obj interface{}) error {
	result, err := c.sendRequest("", http.MethodGet, query, obj)
	if err != nil {
		return err
	}
	if result.State.Code != 0 {
		return errors.New(result.State.Msg)
	}
	var tmp web.ListData
	b, err := json.Marshal(result.Data)
	if err != nil {
		return fmt.Errorf("marshal data with error: %v", err)
	}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return fmt.Errorf("unmarshal data with error: %v", err)
	}
	tmpb, err := json.Marshal(tmp.Items)
	if err != nil {
		return fmt.Errorf("marshal items with error: %v", err)
	}
	if err := json.Unmarshal(tmpb, obj); err != nil {
		return fmt.Errorf("unmarshal items with error: %v", err)
	}
	return nil
}

func (c *clientset) sendRequest(id, method string, query *web.Request, obj interface{}) (*web.Response, error) {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.manager+c.managerURL(id, method, obj), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	// req build query
	if query != nil {
		q := req.URL.Query()
		q.Add("deleted", fmt.Sprintf("%v", query.Deleted))
	}
	req.Header.Add("Authorization", "Bearer "+c.token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result web.Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *clientset) managerURL(id, m string, obj interface{}) string {
	var u string
	u = c.urlSuffix(obj)
	switch m {
	case http.MethodGet:
		if u[len(u)-1] == '/' {
			u = u + id
		}
		return u
	case http.MethodPut, http.MethodDelete:
		u = u + id
	}
	return u
}

func (c *clientset) urlSuffix(obj interface{}) string {
	switch obj.(type) {
	case *module.Module:
		return "/api/v1/module/"
	case *[]module.Module:
		return "/api/v1/module"
	case *project.Project:
		return "/api/v1/project/"
	case *[]project.Project:
		return "/api/v1/project"
	case *cluster.Cluster:
		return "/api/v1/cluster/"
	case *[]cluster.Cluster:
		return "/api/v1/cluster"
	case *application.Application:
		return "/api/v1/application/"
	case *[]application.Application:
		return "/api/v1/application"

	default:
		return ""
	}
}
