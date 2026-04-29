package competition

import "fmt"

// SafeCall runs fn and recovers from panics, returning them as errors.
// Use this to wrap adapter calls so that a single crashing library
// does not kill the entire pipeline run.
func SafeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

// SafeCallValue runs fn and recovers from panics. On panic the zero
// value of T is returned along with the panic as an error.
func SafeCallValue[T any](fn func() (T, error)) (val T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			val = zero
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}
