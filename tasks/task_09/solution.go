package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	if workers > len(in) {
		workers = len(in)
	}
	results := make([]Result[R], len(in))
	jobs := make(chan int)
	wg := sync.WaitGroup{}
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case i, ok := <-jobs:
					if !ok {
						return
					}

					value, err := fn(ctx, in[i])
					if err != nil {
						results[i].Err = err
					} else {
						results[i].Value = value
					}
				}
			}
		}()
	}

outer:
	for i := range in {
		select {
		case <-ctx.Done():
			break outer
		case jobs <- i:
		}
	}

	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
