package errisnil

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

type nilFact int

const (
	nilUnknown nilFact = iota
	nilKnown
	nilMaybe
)

func (n nilFact) String() string {
	switch n {
	case nilUnknown:
		return "nil-unknown"
	case nilKnown:
		return "nil-known"
	case nilMaybe:
		return "nil-maybe"
	default:
		return fmt.Sprintf("nil-unknown(%d)", n)
	}
}

type facts []fact

type fact struct {
	// Union, only one of the below may be set.

	// An error value that is known to be nil (from binary comparison to nil)
	knownNil ssa.Value

	// An error value that is nil in some branches, but not another.
	maybeNil ssa.Value
}

func (fs facts) nilness(v ssa.Value) nilFact {
	for _, f := range fs {
		if f.knownNil == v {
			return nilKnown
		}
		if f.maybeNil == v {
			return nilMaybe
		}
	}

	return nilUnknown
}

func (fs facts) withNilFact(v ssa.Value, n nilFact) facts {
	if n == nilKnown {
		return fs.withKnownNil(v)
	}
	if n == nilMaybe {
		return fs.withMaybeNil(v)
	}
	panic("shouldn't add unknown nil fact")
}

func (fs facts) withKnownNil(v ssa.Value) facts {
	if v == nil {
		panic(v)
	}
	return append(fs, fact{knownNil: v})
}

func (fs facts) withMaybeNil(v ssa.Value) facts {
	if v == nil {
		panic(v)
	}
	return append(fs, fact{maybeNil: v})
}
