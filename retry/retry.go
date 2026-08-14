package retry

import (
	"context"
	"errors"
)

const defaultRetryCount = 5

var ErrBadRequest = errors.New("some 400 status error")

func Retry(ctx context.Context, handler func() (any, error)) (result any, err error) {
	for i := 0; i < defaultRetryCount; i++ {
		result, err = handler()
		if err != nil {
			if errors.Is(err, ErrBadRequest) {
				return nil, err
			}
		}
		if err == nil {
			break
		}
	}

	return result, nil
}
