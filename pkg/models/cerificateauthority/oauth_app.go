package cerificateauthority

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hex-techs/rocket/pkg/models"
	"github.com/hex-techs/rocket/pkg/utils/storage"
	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm"
)

// Application the app object
type Application struct {
	models.Base
	// 应用名称
	Name string `gorm:"NOT NULL;UNIQUE;INDEX;SIZE:64" json:"name,omitempty"`
	// 应用中文名称
	DisplayName string `gorm:"NOT NULL;UNIQUE;INDEX;SIZE:128" json:"displayName,omitempty"`
	// 描述信息
	Description string `gorm:"SIZE:256" json:"description,omitempty"`
	// 自动生成的 id
	SecretID string `gorm:"NOT NULL;UNIQUE;INDEX;SIZE:64" json:"secretId,omitempty"`
	// 应用的 key
	SecretKey string `gorm:"NOT NULL;UNIQUE;INDEX;SIZE:256" json:"secretKey,omitempty"`
	// 跳转地址
	RedirectURI string `gorm:"SIZE:256" json:"redirectUri,omitempty"`
	// 是否更新 key
	UpdateKey bool `gorm:"-" json:"updateKey,omitempty"`
}

// BeforeCreate befor create hook
func (a *Application) BeforeCreate(_ *gorm.DB) error {
	a.secretID()
	a.secretKey()
	return nil
}

// 生成 application 的 secret id
func (a *Application) secretID() {
	id, _ := uuid.NewUUID()
	a.SecretID = id.String()
}

// 生成 application 的 secret key
func (a *Application) secretKey() {
	sig := fmt.Sprintf("%s%s%d", a.SecretID, salt, time.Now().Unix())
	key := pbkdf2.Key([]byte(sig), []byte(salt), 10000, 50, sha256.New)
	a.SecretKey = fmt.Sprintf("%x", key)
}

// list application
func GetAppList(ctx context.Context, db storage.Engine) ([]Application, error) {
	var apps []Application
	_, err := db.List(ctx, 1000, 1, "", &apps)
	return apps, err
}

// 根据 secret key 获取 application
func GetApplicationByKey(ctx context.Context, db storage.Engine, key string) (*Application, error) {
	var app Application
	dbclient := db.Client()
	cli, _ := dbclient.(*gorm.DB)
	c := cli.WithContext(ctx).Where("secret_key= ?", key).First(&app)
	return &app, c.Error
}

// GetAppWithSQL get app info with given sql
func GetAppWithSQL(ctx context.Context, db storage.Engine, sql string, e interface{}) (*Application, error) {
	var app Application
	dbclient := db.Client()
	cli, _ := dbclient.(*gorm.DB)
	c := cli.WithContext(ctx).Where(sql, e).First(&app)
	return &app, c.Error
}

// UpdateApplication update application
func UpdateApplication(ctx context.Context, db storage.Engine, id uint, app *Application) error {
	var oldApp Application
	err := db.Get(ctx, id, "", false, &oldApp)
	if err != nil {
		return err
	}
	app.ID = id
	if app.UpdateKey {
		app.secretID()
		app.secretKey()
	} else {
		app.SecretID = oldApp.SecretID
		app.SecretKey = oldApp.SecretKey
	}
	return db.ForceUpdate(ctx, id, "", &oldApp, app)
}
