package authentication

const (
	// 账号不存在
	ErrAccountNotFound = "account not found"
	// 无法生成token
	ErrGenerateToken = "generate token failed"
	// 密码错误
	ErrPasswordInvalid = "password invalid"
	// 邮箱不允许注册
	ErrEmailNotAllowed = "email not allowed"
	// 注册失败
	ErrRegisterFailed = "register failed"
	// 修改密码失败
	ErrChangePasswordFailed = "change password failed"
	// 发送邮件失败
	ErrSendResetEmailFailed = "send reset email failed"
	// 重置密码token无效
	ErrResetTokenInvalid = "reset token invalid"
	// 重置密码失败
	ErrResetPasswordFailed = "reset password failed"
	// 无效的参数
	ErrInvalidParam = "invalid param"
	// 其他错误
	ErrOther = "other error"
)

var errorMap = map[string]int{
	ErrAccountNotFound:      10001,
	ErrGenerateToken:        10002,
	ErrPasswordInvalid:      10003,
	ErrEmailNotAllowed:      10004,
	ErrRegisterFailed:       10005,
	ErrSendResetEmailFailed: 10006,
	ErrResetTokenInvalid:    10007,
	ErrResetPasswordFailed:  10008,
	ErrChangePasswordFailed: 10009,
	ErrInvalidParam:         10010,
	ErrOther:                10011,
}
