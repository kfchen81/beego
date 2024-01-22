package vanilla

const ERROR_TYPE_BUSINESS = 0
const ERROR_TYPE_SYSTEM = 1

type BusinessError struct {
	Type int
	ErrCode string
	ErrMsg string
}

func (this *BusinessError) Error() string {
	return this.ErrCode
}

func (this *BusinessError) IsPanicError() bool {
	return this.Type == ERROR_TYPE_SYSTEM
}

func NewBusinessError(code string, msg string) *BusinessError {
	return &BusinessError{
		ERROR_TYPE_BUSINESS,
		code,
		msg,
	}
}

func NewSystemError(code string, msg string) *BusinessError {
	return &BusinessError{
		ERROR_TYPE_SYSTEM,
		code,
		msg,
	}
}

