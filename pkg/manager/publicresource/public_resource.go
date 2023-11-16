package publicresource

import (
	"context"
	"net/http"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/models/publicresource"
	"github.com/hex-techs/rocket/pkg/utils/errs"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

func NewPublicResourceController(s *storage.Engine) web.RestController {
	return &PublicResourceController{
		store: s,
	}
}

type PublicResourceController struct {
	web.DefaultController
	store *storage.Engine
}

// Name implements the Name method of RestController
func (*PublicResourceController) Name() string {
	return "publicresource"
}

// Create implements the Create method of RestController
func (p *PublicResourceController) Create() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var pr publicresource.PublicResource
		if err := c.ShouldBindJSON(&pr); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		log.Debugw("create new publicresource", "publicresource", pr)
		if err := p.store.Create(c, &pr); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrCreated, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Delete implements the Delete method of RestController
func (p *PublicResourceController) Delete() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		log.Debugw("delete publicresource", "publicresource_id", id)
		if err := p.store.Delete(c, id, "", &publicresource.PublicResource{}); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Update implements the Update method of RestController
func (p *PublicResourceController) Update() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var pr publicresource.PublicResource
		if err := c.ShouldBindJSON(&pr); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		log.Debugw("update publicresource", "publicresource", pr)
		if err := p.store.Update(context.Background(), id, "", &pr, &pr); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Get implements the Get method of RestController
func (p *PublicResourceController) Get() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var pr publicresource.PublicResource
		if err := p.store.Get(context.Background(), id, "", false, &pr); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
			return
		}
		c.JSON(http.StatusOK, web.DataResponse(pr))
	}, nil
}

// List implements the List method of RestController
func (p *PublicResourceController) List() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var req web.Request
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		req.Default()
		var prs []publicresource.PublicResource
		total, err := p.store.List(context.Background(), req.Limit, req.Page, "", &prs)
		if err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrList, err))
			return
		}
		c.JSON(http.StatusOK, web.ListResponse(int(total), prs))
	}, nil
}

// Middlewares implements the Middlewares method of RestController
func (p *PublicResourceController) Middlewares() []web.MiddlewaresObject {
	return []web.MiddlewaresObject{
		{
			Methods: []string{web.CREATE, web.DELETE, web.UPDATE, web.GET, web.LIST},
			// Middlewares: []gin.HandlerFunc{web.LoginRequired()},
		},
	}
}
