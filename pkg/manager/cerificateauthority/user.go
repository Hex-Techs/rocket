package cerificateauthority

import (
	"context"
	"net/http"

	"github.com/fize/go-ext/log"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/models/cerificateauthority"
	"github.com/hex-techs/rocket/pkg/utils/errs"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

// UserController user controller
type UserController struct {
	web.DefaultController
	Store *storage.Engine
}

// NewUserController return a new user controller
func NewUserController(s *storage.Engine) web.RestController {
	return &UserController{
		Store: s,
	}
}

// 资源名
func (*UserController) Name() string {
	return "user"
}

// Create 创建新用户，只有管理员才能创建
func (uc *UserController) Create() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var u cerificateauthority.User
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		u.EncodePasswd()
		if err := uc.Store.Create(context.TODO(), &u); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrCreated, err))
			return
		}
		log.Debugw("create user", "user", u)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Delete 删除用户，只有管理员才能删除
func (uc *UserController) Delete() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		u := web.GetCurrentUser(c)
		// 不能删除自己
		if u.ID == id {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, errs.ErrDeleteSelf))
			return
		}
		if err := uc.Store.Delete(context.TODO(), id, "", &cerificateauthority.User{}); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrDeleted, err))
			return
		}
		log.Debugw("delete user", "id", id, "optionUserId", u.ID)
		c.JSON(http.StatusOK, web.OkResponse())
	}, nil
}

// Update 管理员普通用户都可以
func (uc *UserController) Update() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		var u cerificateauthority.User
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		cu := web.GetCurrentUser(c)
		if u.ID != id {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrUpdated, errs.ErrUpdateOther))
			return
		}
		if err := uc.Store.Update(context.TODO(), id, "", &cerificateauthority.User{}, &u); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, err))
			return
		}
		log.Debugw("update user", "id", id, "user", u, "current user", cu)
		// 刷新用户信息
		if err := u.GenUser(); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrUpdated, errs.ErrFlushUserInfo))
			return
		}
		u.TruncatePassword()
		c.JSON(http.StatusOK, web.DataResponse(u))
	}, nil
}

// Get 获取用户详情
func (uc *UserController) Get() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		id, err := web.GetID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrID, err))
			return
		}
		cu := web.GetCurrentUser(c)
		if cu.ID != id {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrGet, errs.ErrGetOtherUser))
			return
		}
		var u cerificateauthority.User
		if err := uc.Store.Get(context.TODO(), id, "", false, &u); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errs.ErrGet, err))
			return
		}
		u.TruncatePassword()
		c.JSON(http.StatusOK, web.DataResponse(u))
	}, nil
}

// List 获取用户列表
func (uc *UserController) List() (gin.HandlerFunc, error) {
	return func(c *gin.Context) {
		var req web.Request
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrInvalidParam, err))
			return
		}
		req.Default()
		log.Debugf("list user: %v", req)
		var us []cerificateauthority.User
		total, err := uc.Store.List(context.TODO(), req.Limit, req.Page, "", &us)
		if err != nil {
			c.JSON(http.StatusBadRequest, web.ExceptResponse(errs.ErrList, err))
			return
		}
		for i, u := range us {
			u.TruncatePassword()
			us[i] = u
		}
		c.JSON(http.StatusOK, web.ListResponse(int(total), us))
	}, nil
}

func (uc *UserController) Middlewares() []web.MiddlewaresObject {
	return []web.MiddlewaresObject{
		{
			Methods: []string{web.CREATE, web.DELETE, web.UPDATE, web.GET, web.LIST},
			// Middlewares: []gin.HandlerFunc{web.LoginRequired()},
		},
	}
}
