package retry

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

var errTransient = errors.New("transient error")
var errPermanent = errors.New("permanent error")

func alwaysRetry(_ error) bool { return true }
func neverRetry(_ error) bool  { return false }

func TestRetry(t *testing.T) {
	type args struct {
		ctx         context.Context
		shouldRetry func(error) bool
		handler     func() (any, error)
	}
	tests := []struct {
		name       string
		args       args
		wantResult any
		wantErr    bool
	}{
		{
			name: "success on first call",
			args: args{
				ctx:         context.Background(),
				shouldRetry: alwaysRetry,
				handler:     func() (any, error) { return "ok", nil },
			},
			wantResult: "ok",
			wantErr:    false,
		},
		{
			name: "success after transient failures",
			args: args{
				ctx:         context.Background(),
				shouldRetry: alwaysRetry,
				handler: func() func() (any, error) {
					calls := 0
					return func() (any, error) {
						calls++
						if calls < 3 {
							return nil, errTransient
						}
						return "ok", nil
					}
				}(),
			},
			wantResult: "ok",
			wantErr:    false,
		},
		{
			name: "non-retryable error stops immediately",
			args: args{
				ctx:         context.Background(),
				shouldRetry: neverRetry,
				handler:     func() (any, error) { return nil, errPermanent },
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name: "all retries exhausted returns error",
			args: args{
				ctx:         context.Background(),
				shouldRetry: alwaysRetry,
				handler:     func() (any, error) { return nil, errTransient },
			},
			wantResult: nil,
			wantErr:    true, // fails with current code: final return ignores err
		},
		{
			name: "cancelled context stops retries",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				shouldRetry: alwaysRetry,
				handler:     func() (any, error) { return nil, errTransient },
			},
			wantResult: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := Retry(tt.args.ctx, tt.args.shouldRetry, tt.args.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("Retry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("Retry() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
