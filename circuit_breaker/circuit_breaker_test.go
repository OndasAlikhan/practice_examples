package circuit_breaker

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type tT struct {
	msg string
}

func TestCircuitBreaker_Run(t *testing.T) {

	type args[T any] struct {
		handler Handler[T]
	}
	type testCase[T any] struct {
		name       string
		cb         *CircuitBreaker[T]
		args       args[T]
		wantResult T
		wantErr    error
	}
	tests := []testCase[tT]{
		{
			name: "success",
			cb:   New[tT](),
			args: args[tT]{
				handler: func() (tT, error) {
					return tT{
						msg: "success",
					}, nil
				},
			},
			wantResult: tT{
				msg: "success",
			},
			wantErr: nil,
		},
		{
			name: "open",
			cb:   New[tT](),
			args: args[tT]{
				handler: func() (tT, error) {
					return tT{
						msg: "fail",
					}, errors.New("failed")
				},
			},
			wantResult: tT{},
			wantErr:    ErrOpenState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotResult tT
			var err error
			for _ = range 5 {
				gotResult, err = tt.cb.Run(tt.args.handler)
				if err != nil && errors.Is(err, ErrOpenState) {
					break
				}
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Run() error = %v, tt.wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("Run() gotResult = %v, want = %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestCircuitBreaker_RunHalfOpenThenOpen(t *testing.T) {
	cb := New[tT]()
	count := 0
	var mt sync.Mutex

	handler := func() (tT, error) {
		mt.Lock()
		defer mt.Unlock()
		defer func() {
			count += 1
		}()
		if count >= 3 {
			return tT{}, errors.New("failed")
		}
		return tT{msg: "success"}, nil
	}

	var err error

	g, _ := errgroup.WithContext(context.Background())

	for gIndex := range 3 {
		g.Go(func() error {
			var gErr error
			for _ = range 7 {
				_, gErr = cb.Run(handler)
				fmt.Printf("gIndex: %d, gErr: %v\n", gIndex, gErr)
				if gErr != nil && errors.Is(gErr, ErrOpenState) {
					break
				}
			}
			return gErr
		})
	}
	err = g.Wait()

	require.ErrorIs(t, err, ErrOpenState)
	require.Equal(t, open, cb.state)

	if errors.Is(err, ErrOpenState) {
		time.Sleep(time.Second * 35)
	}

	_, singleErr := cb.Run(handler)

	require.ErrorIs(t, singleErr, ErrOpenState)
	require.Equal(t, open, cb.state)
}

func TestCircuitBreaker_RunHalfOpenThenClosed(t *testing.T) {
	cb := New[tT]()
	count := 0

	handler := func() (tT, error) {
		defer func() {
			count += 1
		}()
		if count >= 3 {
			return tT{}, errors.New("failed")
		}
		return tT{msg: "success"}, nil
	}

	successHandler := func() (tT, error) {
		return tT{msg: "success"}, nil
	}

	var result tT
	var err error

	g, _ := errgroup.WithContext(context.Background())

	for gIndex := range 3 {
		g.Go(func() error {
			var gErr error
			for _ = range 7 {
				result, gErr = cb.Run(handler)
				fmt.Printf("gIndex: %d, gErr: %v\n", gIndex, gErr)
				if gErr != nil && errors.Is(gErr, ErrOpenState) {
					break
				}
			}
			return gErr
		})
	}

	err = g.Wait()
	require.ErrorIs(t, err, ErrOpenState)
	require.Equal(t, open, cb.state)

	if errors.Is(err, ErrOpenState) {
		time.Sleep(time.Second * 35)
	}

	g2, _ := errgroup.WithContext(context.Background())

	for gIndex := range 3 {
		g2.Go(func() error {
			var gErr error
			for _ = range 7 {
				result, gErr = cb.Run(successHandler)
				fmt.Printf("gIndex: %d, gErr: %v\n", gIndex, gErr)
				if gErr != nil && errors.Is(gErr, ErrOpenState) {
					break
				}
			}
			return gErr
		})
	}

	secondErr := g2.Wait()
	require.NoError(t, secondErr)
	require.Equal(t, "success", result.msg)
	require.Equal(t, closed, cb.state)
}
