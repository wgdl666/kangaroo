package bizerror

import (
	"errors"
	"fmt"
)

func As(err error) (BizError, bool) {
	// var causeErr BizError
	// 先限定为核心包实现类型.
	var causeErr *bizErrorImpl
	if errors.As(err, &causeErr) {
		return causeErr, true
	}
	return nil, false
}

// ChainWith 对 err 信息补充, 构建chain 信息链.
// 如果 err 实现 BizError 接口, 则记录于 chain .
// 反之, 以 fmt.Errorf 构造.
func ChainWith(err error, format string, args ...any) error {
	bizErr, ok := err.(BizError)
	if !ok {
		details := fmt.Sprintf(format, args...)
		return fmt.Errorf("with=[%s] err=%w", details, err)
	}
	return bizErr.Errorf(format, args...)
}

// ParseCode 尝试对 err 解析为 BizError, 返回对应 (Code, true), 反之返回 (0, false).
func ParseCode(err error) (code int32, ok bool) {
	bizErr, ok := As(err)
	if !ok {
		return 0, false
	}
	return bizErr.Code(), true
}

func ParseMessage(err error) string {
	bizErr, ok := As(err)
	if !ok {
		return err.Error()
	}
	return bizErr.Message()
}
