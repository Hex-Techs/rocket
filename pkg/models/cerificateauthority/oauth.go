package cerificateauthority

import (
	"context"

	"github.com/hex-techs/rocket/pkg/utils/storage"
	"gorm.io/gorm"
)

// Grant A grant code is created in the authorization flow,
// and will be destroyed when the authorization is finished.
type Grant struct {
	ID      uint   `gorm:"primary_key;index"`
	UserID  uint   `gorm:"index"`
	AppID   uint   `gorm:"index"`
	Code    string `gorm:"size:128"`
	Expires int64
}

// GetGrantWithSQL get grant info with given sql
func GetGrantWithSQL(ctx context.Context, db storage.Engine, sql string, e interface{}) (*Grant, error) {
	var grant Grant
	dbclient := db.Client()
	cli, _ := dbclient.(*gorm.DB)
	c := cli.WithContext(ctx).Where(sql, e).First(&grant)
	return &grant, c.Error
}

// Token A bearer token is the final token that could be used by the client.
// Bearer token is widely used. Liuer only comes with a bearer token.
type Token struct {
	ID          uint   `gorm:"primary_key;index"`
	UserID      uint   `gorm:"index"`
	AppID       uint   `gorm:"index"`
	AccessToken string `gorm:"size:128"`
	ExpiredIn   int64
}

// GetTokenWithSQL get token info with given sql
func GetTokenWithSQL(ctx context.Context, db storage.Engine, sql string, e interface{}) (*Token, error) {
	var token Token
	dbclient := db.Client()
	cli, _ := dbclient.(*gorm.DB)
	c := cli.WithContext(ctx).Where(sql, e).First(&token)
	return &token, c.Error
}
