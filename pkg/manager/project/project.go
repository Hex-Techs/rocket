package project

import (
	"net/http"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/models/project"
	"github.com/hex-techs/rocket/pkg/utils/errs"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

// ProjectController project controller
type ProjectController struct {
	web.DefaultController
	Store *storage.Engine
}

// NewProjectController return a new project controller
func NewProjectController(s *storage.Engine) web.RestController {
	return &ProjectController{
		Store: s,
	}
}

// 资源名
func (*ProjectController) Name() string {
	return "project"
}

// Create 创建新项目
func (pc *ProjectController) Create() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var project project.Project
		if err := c.ShouldBindJSON(&project); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		log.Debugf("create project: %v", project)
		if err := pc.Store.Create(c, &project); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrCreated, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Delete 删除项目
func (pc *ProjectController) Delete() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		if err := pc.Store.ForceDelete(c, id, "", project.Project{}); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Update 更新项目信息
func (pc *ProjectController) Update() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var new project.Project
		c.ShouldBindJSON(&new)
		if err := pc.Store.Update(c, id, "", &project.Project{}, &new); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Get 获取项目详情
func (pc *ProjectController) Get() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var project project.Project
		if err := pc.Store.Get(c, id, "", false, &project); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
			return
		}
		c.JSON(http.StatusOK, web.DataResponse(project))
	}, nil
}

// List 获取项目列表
func (pc *ProjectController) List() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var req web.Request
		c.ShouldBindQuery(&req)
		req.Default()
		log.Debugf("list projects: %+v", req)
		var projects []project.Project
		total, err := pc.Store.List(c, req.Limit, req.Page, "", &projects)
		if err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrList, err))
			return
		}
		c.JSON(http.StatusOK, web.ListResponse(int(total), projects))
	}, nil
}

func (uc *ProjectController) Middlewares() []web.MiddlewaresObject {
	return []web.MiddlewaresObject{
		{
			Methods: []string{web.CREATE, web.DELETE, web.UPDATE, web.GET, web.LIST},
			// Middlewares: []gin.HandlerFunc{web.LoginRequired()},
		},
	}
}
