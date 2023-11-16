package authentication

// LoginForm
type LoginForm struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ForgetPasswordForm
type ForgetPasswordForm struct {
	Name string `json:"name" binding:"required"`
}

// ChangePasswordForm
type ChangePasswordForm struct {
	OldPassword        string `json:"oldPassword" binding:"required"`
	NewPassword        string `json:"newPassword" binding:"required,nefield=OldPassword,min=6"`
	NewPasswordConfirm string `json:"newPasswordConfirm" binding:"eqfield=NewPassword"`
}

// ResetPasswordForm
type ResetPasswordForm struct {
	Password        string `json:"password" binding:"required,min=6"`
	PasswordConfirm string `json:"passwordConfirm" binding:"eqfield=Password"`
	Token           string `json:"token" binding:"required"`
}

// RegisterForm
type RegisterForm struct {
	Email       string `json:"email" binding:"required,email"`
	Name        string `json:"name" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
	Password2   string `json:"password2" binding:"eqfield=Password"`
	DisplayName string `json:"displayName" binding:"required"`
	Phone       string `json:"phone"`
	IM          string `json:"im"`
}
