package vanilla

const ERROR_TYPE_BUSINESS = 0
const ERROR_TYPE_SYSTEM = 1

type BusinessError struct {
	Type             int
	ErrCode          string
	ErrMsg           string
	needPushToSentry bool // 是否需要推送给sentry, 默认为true
}

func (this *BusinessError) Error() string {
	return this.ErrCode
}

func (this *BusinessError) IsPanicError() bool {
	return this.Type == ERROR_TYPE_SYSTEM
}

// NoPush 设置不需要推给 sentry
func (this *BusinessError) NoPush() *BusinessError {
	this.needPushToSentry = false
	return this
}

// IsNeedPush 是否需要推送给 sentry
func (this *BusinessError) IsNeedPush() bool {
	return this.needPushToSentry || this.IsPanicError()
}

func NewBusinessError(code string, msg string) *BusinessError {
	return &BusinessError{
		ERROR_TYPE_BUSINESS,
		code,
		msg,
		true,
	}
}

func NewBusinessErrorFromError(err error) *BusinessError {
	switch err.(type) {
	case *BusinessError:
		return err.(*BusinessError)
	}

	errCode := err.Error()
	return &BusinessError{
		ERROR_TYPE_BUSINESS,
		errCode,
		errCode,
		true,
	}
}

func NewSystemError(code string, msg string) *BusinessError {
	return &BusinessError{
		ERROR_TYPE_SYSTEM,
		code,
		msg,
		true,
	}
}
