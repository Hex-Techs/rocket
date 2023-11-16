package cerificateauthority

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-faker/faker/v4"
	"github.com/hex-techs/blade/pkg/utils/config"
	"github.com/hex-techs/blade/pkg/utils/token"
	"github.com/hex-techs/rocket/pkg/models"
	"golang.org/x/crypto/pbkdf2"
)

const salt = "rocket"

// User
type User struct {
	models.Base
	// Name user name, unique, not null
	Name string `gorm:"unique,index,size:64,not null" json:"name,omitempty" binding:"required"`
	// Display name
	DisplayName string `gorm:"size:64" json:"displayName,omitempty" binding:"required"`
	// Password user password
	Password string `gorm:"size:1024" json:"password,omitempty"`
	// Email user email, unique, not null
	Email string `gorm:"unique,index,not null" json:"email,omitempty" binding:"required"`
	// Enabled user enabled
	Enabled bool `gorm:"default:true" json:"enabled,omitempty"`
	// Phone user phone
	Phone string `gorm:"size:32" json:"phone,omitempty"`
	// user im
	IM string `gorm:"size:128" json:"im,omitempty"`
	// token
	Token *UserToken `gorm:"-" json:"token,omitempty"`
}

// Token response user's token
type UserToken struct {
	// token信息
	Token string `json:"token"`
	// 过期时间
	Expired int64 `json:"expired"`
}

// EncodePasswd encodes password to safe format.
func (u *User) EncodePasswd() {
	newPasswd := pbkdf2.Key([]byte(u.Password), []byte(salt), 10000, 50, sha256.New)
	u.Password = fmt.Sprintf("%x", newPasswd)
}

// ValidatePassword checks if given password matches the one belongs to the user.
func (u *User) ValidatePassword(Password string) bool {
	newUser := &User{Password: Password}
	newUser.EncodePasswd()
	return subtle.ConstantTimeCompare([]byte(u.Password), []byte(newUser.Password)) == 1
}

// GenUser generate User
func (u *User) GenUser() error {
	claim := &token.Claims{
		ID:             u.ID,
		Name:           u.Name,
		StandardClaims: jwt.StandardClaims{},
	}
	t, e, err := token.GenerateJWTToken(claim, config.Read().Service.TokenExpired)
	if err != nil {
		return err
	}
	u.Token = &UserToken{
		Token:   t,
		Expired: e,
	}
	return nil
}

// TruncatePassword return null password to client
func (u *User) TruncatePassword() {
	u.Password = ""
}

func FakeData(count int) []*User {
	var res []*User
	for i := 0; i < count; i++ {
		res = append(res, &User{
			Name:        faker.Username(),
			DisplayName: faker.ChineseFirstName() + faker.ChineseLastName(),
			Password:    "123123",
			Email:       faker.Email(),
			Enabled:     true,
			Phone:       faker.E164PhoneNumber(),
		})
	}
	return res
}
