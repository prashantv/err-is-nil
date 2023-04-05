package test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
)

func maybeErr1() error { return maybeErr() }
func maybeErr2() error { return maybeErr() }
func maybeErr3() error { return maybeErr() }
func maybeErr4() error { return maybeErr() }
func maybeErr5() error { return maybeErr() }

func maybeErr() error {
	if rand.Intn(3) == 1 {
		return errors.New("err")
	}

	return nil
}

func standardErrCheck() error {
	err := maybeErr()
	if err != nil {
		return err
	}

	fmt.Println(err) // want `known to be nil`

	return nil
}

func reverseStandardErrCheck() error {
	err := maybeErr()
	if nil != err {
		return err
	}

	fmt.Println(err) // want `known to be nil`

	return nil
}

func multipleErrCheck() error {
	if err := maybeErr(); err != nil {
		return err
	}

	err := maybeErr()
	if err != nil {
		return err
	}

	err = maybeErr()
	fmt.Println(err) // err was reset, so no longer known as nil.
	if err != nil {
		return err
	}

	return err // want `known to be nil`
}

func errNilCheck() error {
	err := maybeErr()
	if err == nil {
		return err // want `known to be nil`
	}

	err = maybeErr()
	if nil == err {
		return err // want `known to be nil`
	}

	return err
}

func errNilCheckMissingReturn() error {
	err := maybeErr()
	if err != nil {
		if anotherErr := maybeErr(); anotherErr != nil {
			fmt.Println(err)
			return anotherErr
		}

		fmt.Println(err)
	}
	return err
}

func errNested() error {
	err := maybeErr()
	if err != nil {
		if err := maybeErr(); err != nil {
			if err := maybeErr(); err != nil {
				if err := maybeErr(); err != nil {
					return err
				}
				fmt.Println(err)
			} else {
				fmt.Println(err) // want `known to be nil`
			}
			return err
		}
		return err
	}

	return err // want `known to be nil`
}

func multipleErrNotNilCheck() error {
	err := maybeErr()
	if err != nil {
		return err
	}

	if true {
		err = maybeErr()
	}

	if err != nil { // want `nil in some branches`
		if true {
			err = maybeErr()
		}

		fmt.Println(err)
	}

	return nil
}

func newTeamBySlugLoader(ctx context.Context, converter *struct{}, client struct{}) any {
	type loader struct {
		fetch func() []error
	}
	return &loader{
		fetch: func() []error {
			err := maybeErr1()
			if err != nil {
				return []error{err}
			}

			for i := 0; i < 10; i++ {
				err := maybeErr2()
				if err != nil {
					return []error{err}
				}
			}

			errs := make([]error, 10)
			for i := 0; i < 10; i++ {
				if i%2 == 0 {
					errs[i] = errors.New("not found")
				}
			}

			return errs
		},
	}
}
