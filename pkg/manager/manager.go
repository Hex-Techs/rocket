package manager

import (
	"github.com/gin-gonic/gin"
	"github.com/hex-techs/rocket/pkg/manager/application"
	"github.com/hex-techs/rocket/pkg/manager/cerificateauthority"
	"github.com/hex-techs/rocket/pkg/manager/cerificateauthority/authentication"
	"github.com/hex-techs/rocket/pkg/manager/cluster"
	"github.com/hex-techs/rocket/pkg/manager/communication"
	"github.com/hex-techs/rocket/pkg/manager/module"
	"github.com/hex-techs/rocket/pkg/manager/project"
	"github.com/hex-techs/rocket/pkg/manager/publicresource"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"github.com/hex-techs/rocket/pkg/utils/web"
)

func InstallAPI(r *gin.Engine, s *storage.Engine) {
	installCommunicationAPI(r, s)
	installAuthn(r, s)
	installUserAPI(r, s)
	installModuleAPI(r, s)
	installProjectAPI(r, s)
	installClusterAPI(r, s)
	installPublicResourceAPI(r, s)
	installApplicationAPI(r, s)
}

func installCommunicationAPI(r *gin.Engine, s *storage.Engine) {
	comm := communication.NewCommunicationController(s)
	r.GET("/api/v1/communication", comm.Communication)
}

func installAuthn(r *gin.Engine, s *storage.Engine) {
	api := authentication.NewAuthn(s)
	group := r.Group("/api/v1/auth")
	{
		group.POST("/register", api.Register)
		group.POST("/login", api.Login)
		group.POST("/restpasswordrequest", api.ResetPasswordRequest)
		group.PUT("/resetpassword/:token", api.ResetPassword)
		group.PUT("/changepassword", web.LoginRequired(), api.ChangePassword)
	}
}

func installUserAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, cerificateauthority.NewUserController(s))
}

func installModuleAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, module.NewModuleController(s))
}

func installProjectAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, project.NewProjectController(s))

}

func installClusterAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, cluster.NewClusterController(s))
}

func installPublicResourceAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, publicresource.NewPublicResourceController(s))
}

func installApplicationAPI(r *gin.Engine, s *storage.Engine) {
	u := web.RestfulAPI{
		PostParameter: "/:id",
	}
	u.Install(r, application.NewApplicationController(s))
}
