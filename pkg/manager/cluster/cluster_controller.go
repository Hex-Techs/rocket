/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"net/http"
	"strconv"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/errs"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

// NewClusterController return a new cluster controller
func NewClusterController(s *storage.Engine) web.RestController {
	cc := &ClusterController{
		store: s,
	}
	go cc.heartbeatChecker()
	return cc
}

// ClusterController cluster controller
type ClusterController struct {
	web.DefaultController
	store *storage.Engine
}

// ResourceName
func (*ClusterController) Name() string {
	return "cluster"
}

// Create create new cluster
func (cc *ClusterController) Create() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var cls cluster.Cluster
		if err := c.ShouldBindJSON(&cls); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		log.Debugw("create new cluster", "cluster", cls)
		if err := cc.store.Create(c, &cls); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrCreated, err))
			return
		}
		cache.GetCommunicationManager().AddMessage(cache.CreateAction, &cls)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Delete delete cluster, but resource of user created and deployed in this cluster will not be deleted
func (cc *ClusterController) Delete() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		log.Debugw("delete cluster", "cluster_id", id)
		old := &cluster.Cluster{}
		if err := cc.store.Get(c, id, "", false, old); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		if err := cc.store.ForceDelete(c, id, "", &cluster.Cluster{}); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		if err := cc.deleteRevisionManager(old); err != nil {
			log.Errorw("failed to delete revision manager when delete cluster", "error", err, "cluster", old)
		}
		cache.GetCommunicationManager().AddMessage(cache.DeleteAction, old)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Update update cluster
func (cc *ClusterController) Update() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		i := c.Param("id")
		var name string
		id, err := strconv.Atoi(i)
		if err != nil {
			name = i
		}
		var old, cls cluster.Cluster
		if err := c.ShouldBindJSON(&cls); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		if err := cc.store.Get(c, uint(id), name, false, &old); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		log.Debugw("update cluster", "cluster", cls)
		if err := cc.store.Update(c, uint(id), name, &cls, &cls); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		cls.ID = old.ID
		go cc.storeRevisionManager(&old, &cls)
		cache.GetCommunicationManager().AddMessage(cache.UpdateAction, &cls)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Get
func (cc *ClusterController) Get() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		i := c.Param("id")
		var name string
		id, err := strconv.Atoi(i)
		if err != nil {
			name = i
		}
		var cls cluster.Cluster
		if err := cc.store.Get(c, uint(id), name, false, &cls); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
			return
		}
		c.JSON(http.StatusOK, web.DataResponse(&cls))
	}, nil
}

// List
func (cc *ClusterController) List() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var req web.Request
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		req.Default()
		var clss []cluster.Cluster
		total, err := cc.store.List(c, req.Limit, req.Page, "", &clss)
		if err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrList, err))
			return
		}
		c.JSON(http.StatusOK, web.ListResponse(int(total), clss))
	}, nil
}

func (cc *ClusterController) Middlewares() []web.MiddlewaresObject {
	return []web.MiddlewaresObject{
		{
			Methods: []string{web.CREATE, web.DELETE, web.UPDATE, web.GET, web.LIST},
			// Middlewares: []gin.HandlerFunc{web.LoginRequired()},
		},
	}
}
