package errs

const (
	ErrCreated = iota + 20000
	ErrDeleted
	ErrUpdated
	ErrGet
	ErrList
	ErrID
	ErrInvalidParam
	ErrWebsocket
)

const (
	// 生成token失败
	ErrGenerateUserToken = "generate user token failed"
	// 不能删除自身
	ErrDeleteSelf = "can not delete yourself"
	// 不能修改其他用户
	ErrUpdateOther = "can not update other user"
	// flush user info failed
	ErrFlushUserInfo = "flush user info failed"
	// can not get other user
	ErrGetOtherUser = "can not get other user"
)
