package retry

import (
	"context"
	"errors"
)

const defaultRetryCount = 5

var ErrBadRequest = errors.New("some 400 status error")

func Retry(ctx context.Context, shouldRetry func(error) bool, handler func() (any, error)) (result any, err error) {
	for i := 0; i < defaultRetryCount; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		result, err = handler()
		if err == nil {
			return result, nil
		}
		if !shouldRetry(err) {
			return nil, err
		}
	}

	return result, nil
}
