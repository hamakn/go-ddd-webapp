package concurrency

import "context"

// ExecAllOrAbortOnError is helper to exec all functions or abort on first error
func ExecAllOrAbortOnError(ctx context.Context, funcs []func() error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan error)
	jobs := len(funcs)
	done := 0

	for _, f := range funcs {
		f := f
		go func() {
			err := f()
			select {
			case c <- err:
			case <-ctx.Done():
			}
		}()
	}

	for {
		select {
		case err := <-c:
			done++
			if err != nil {
				return err
			}
			if jobs == done {
				return nil
			}
		}
	}
}
