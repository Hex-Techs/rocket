package authentication

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fize/go-ext/log"
	"github.com/fize/go-ext/sendmail"
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/models/cerificateauthority"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/token"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

// Authn
type Authn struct {
	Store *storage.Engine
}

func NewAuthn(s *storage.Engine) *Authn {
	return &Authn{Store: s}
}

// 登录
func (a *Authn) Login(c *gin.Context) {
	var f LoginForm
	var u cerificateauthority.User
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, web.ExceptResponse(errorMap[ErrInvalidParam], err))
		return
	}
	if err := a.Store.Get(context.TODO(), 0, f.Name, false, &u); err != nil {
		if err.Error() != "record not found" {
			c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrOther], err))
			return
		}
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrAccountNotFound], ErrAccountNotFound))
		return
	}
	log.Debugw("login user", "name", f.Name)
	if u.ValidatePassword(f.Password) {
		if err := u.GenUser(); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrGenerateToken], err))
			return
		}
	} else {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrPasswordInvalid], ErrPasswordInvalid))
		return
	}
	u.TruncatePassword()
	c.JSON(http.StatusOK, web.DataResponse(u))
}

// 注册用户
func (a *Authn) Register(c *gin.Context) {
	var f RegisterForm
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, web.ExceptResponse(errorMap[ErrInvalidParam], err))
		return
	}
	l := strings.Split(f.Email, "@")
	company := l[1]
	if config.Read().Manager.CompanyPost != "" {
		if company != config.Read().Manager.CompanyPost {
			c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrEmailNotAllowed], ErrEmailNotAllowed))
			return
		}
	}
	user := cerificateauthority.User{
		Name:        f.Name,
		Email:       f.Email,
		DisplayName: f.DisplayName,
		Password:    f.Password,
		Phone:       f.Phone,
		IM:          f.IM,
	}
	user.EncodePasswd()
	if err := a.Store.Create(context.TODO(), &user); err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrRegisterFailed], err))
	} else {
		log.Debugw("register user", "user", f)
		c.JSON(http.StatusOK, web.OkResponse())
	}
}

// 修改密码
func (a *Authn) ChangePassword(c *gin.Context) {
	var f ChangePasswordForm
	var u cerificateauthority.User
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, web.ExceptResponse(errorMap[ErrInvalidParam], err))
		return
	}
	cu := web.GetCurrentUser(c)
	if err := a.Store.Get(context.TODO(), cu.ID, cu.Name, false, &u); err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrChangePasswordFailed], err))
		return
	}
	if u.ValidatePassword(f.OldPassword) {
		u.Password = f.NewPassword
		u.EncodePasswd()
		if err := a.Store.Update(context.TODO(), u.ID, u.Name, &u, &u); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrChangePasswordFailed], err))
			return
		}
	} else {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrPasswordInvalid], ErrPasswordInvalid))
		return
	}
	log.Debugw("change password", "user", cu)
	c.JSON(http.StatusOK, web.OkResponse())
}

// 请求重置密码
func (a *Authn) ResetPasswordRequest(c *gin.Context) {
	var f ForgetPasswordForm
	var u cerificateauthority.User
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, web.ExceptResponse(errorMap[ErrInvalidParam], err))
		return
	}
	log.Debugw("reset password request", "user", f)
	if err := a.Store.Get(context.TODO(), 0, f.Name, false, &u); err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrAccountNotFound], err))
		return
	}
	mc := config.Read().Email
	if mc != nil && mc.Account != "" {
		body := fmt.Sprintf("Reset password link: %s%s/%s, valid for %d seconds",
			config.Read().Manager.Domain, config.Read().Manager.ResetPath,
			token.GenerateCustomToken(f.Name, int(config.Read().Manager.URLExpired)), config.Read().Manager.URLExpired)
		if err := sendmail.SendEmail("rocket", mc.SMTP, mc.Account, mc.Password, u.Email,
			"Reset Password", body, mc.Port); err != nil {
			c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrSendResetEmailFailed], err))
			return
		}
	} else {
		c.JSON(http.StatusOK,
			web.DataResponse("Email service is not enabled, please contact the administrator to reset the password!"))
		return
	}
}

// 重置密码
func (a *Authn) ResetPassword(c *gin.Context) {
	var f ResetPasswordForm
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, web.ExceptResponse(errorMap[ErrInvalidParam], err))
		return
	}
	log.Debugf("reset password: %v", f)
	if f.Token == "" {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrResetTokenInvalid], ErrResetTokenInvalid))
		return
	}
	name, err := token.ParseCustomToken(f.Token)
	if err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrResetTokenInvalid], err))
		return
	}
	var u cerificateauthority.User
	if err := a.Store.Get(context.TODO(), 0, name, false, &u); err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrResetPasswordFailed], err))
		return
	}
	u.Password = f.Password
	u.EncodePasswd()
	if err := a.Store.Update(context.TODO(), u.ID, u.Name, &u, &u); err != nil {
		c.JSON(http.StatusOK, web.ExceptResponse(errorMap[ErrResetPasswordFailed], err))
		return
	}
	c.JSON(http.StatusOK, web.OkResponse())
}
