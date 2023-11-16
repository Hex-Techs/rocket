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

package application

import (
	"fmt"
	"net/http"
	"strconv"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/cache"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/utils/errs"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

var kindSet = mapset.NewSet[application.WorkloadType](application.DeploymentType,
	application.CloneSetType, application.CronJobType)

// NewApplicationController return a new application controller
func NewApplicationController(s *storage.Engine) web.RestController {
	ac := &ApplicationController{
		store: s,
	}
	return ac
}

// ApplicationController application controller
type ApplicationController struct {
	web.DefaultController
	store *storage.Engine
}

// ResourceName
func (*ApplicationController) Name() string {
	return "application"
}

// Create create new application
func (ac *ApplicationController) Create() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var app application.Application
		if err := c.ShouldBindJSON(&app); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		if !kindSet.Contains(app.Template.Type) {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam,
				fmt.Sprintf("kind is not supported, only support %s", kindSet.String())))
			return
		}
		log.Debugw("create new application", "application", app)
		if err := ac.store.Create(c, &app); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrCreated, err))
			return
		}
		cache.GetCommunicationManager().AddMessage(cache.CreateAction, app)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Delete delete application
func (ac *ApplicationController) Delete() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		log.Debugw("delete application", "application_id", id)
		old := &application.Application{}
		if err := ac.store.Get(c, id, "", false, old); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		if err := ac.store.Delete(c, id, "", &application.Application{}); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		if err := ac.deleteRevisionManager(old); err != nil {
			log.Errorw("failed to delete revision manager when delete application", "error", err, "application", old)
		}
		cache.GetCommunicationManager().AddMessage(cache.DeleteAction, old)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Update update application
func (ac *ApplicationController) Update() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var old, app application.Application
		if err := c.ShouldBindJSON(&app); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		if !kindSet.Contains(app.Template.Type) {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam,
				fmt.Sprintf("kind is not supported, only support %s", kindSet.String())))
			return
		}
		if err := ac.store.Get(c, id, "", false, &old); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		log.Debugw("update application", "application", app)
		if err := ac.store.Update(c, id, "", &app, &app); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		app.ID = id
		go ac.storeRevisionManager(&old, &app)
		cache.GetCommunicationManager().AddMessage(cache.UpdateAction, app)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Get
func (ac *ApplicationController) Get() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		i := c.Param("id")
		var name string
		id, err := strconv.Atoi(i)
		if err != nil {
			name = i
		}
		var req web.Request
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		var app application.Application
		if req.Deleted {
			if err := ac.store.Get(c, uint(id), name, true, &app); err != nil {
				c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
				return
			}
		} else {
			if err := ac.store.Get(c, uint(id), name, false, &app); err != nil {
				c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
				return
			}
		}
		c.JSON(http.StatusOK, web.DataResponse(app))
	}, nil
}

// List
func (ac *ApplicationController) List() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var req web.Request
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		req.Default()
		var apps []application.Application
		total, err := ac.store.List(c, req.Limit, req.Page, "", &apps)
		if err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrList, err))
			return
		}
		c.JSON(http.StatusOK, web.ListResponse(int(total), apps))
	}, nil
}

func (ac *ApplicationController) Middlewares() []web.MiddlewaresObject {
	return []web.MiddlewaresObject{
		{
			Methods: []string{web.CREATE, web.DELETE, web.UPDATE, web.GET, web.LIST},
			// Middlewares: []gin.HandlerFunc{web.LoginRequired()},
		},
	}
}
