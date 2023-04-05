package errisnil

import (
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

const Doc = `check for unnecessary use of error variables that are known to be nil

The err-is-nil checker looks for code following an error != nil return
that still uses the error variable, even though it's known to be nil.
`

var Analyzer = &analysis.Analyzer{
	Name:     "errisnil",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	ssainput := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	runner := runner{
		pass:    pass,
		visited: make(map[*ssa.BasicBlock]bool),
	}
	for _, fn := range ssainput.SrcFuncs {
		debug("visit function", fn.Name())
		runner.visit(fn.Blocks[0], make(facts, 0, 20))
	}
	return nil, nil
}

type runner struct {
	pass    *analysis.Pass
	visited map[*ssa.BasicBlock]bool
}

func (r runner) report(pos token.Pos, format string, args ...interface{}) {
	r.pass.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: "errisnil",
		Message:  fmt.Sprintf(format, args...),
	})
}

func (r runner) visit(b *ssa.BasicBlock, fs facts) {
	if r.visited[b] {
		return
	}
	r.visited[b] = true

	var operands []*ssa.Value

	debug("  visit block", withType(b))
	for _, instr := range b.Instrs {
		debug("    instr", withType(instr))
		operands = instr.Operands(operands[:0])

		if phi, ok := instr.(*ssa.Phi); ok {
			switch visitPhi(fs, operands) {
			case nilKnown:
				debug("    add knownNil and visit", withType(phi))
				fs = fs.withKnownNil(phi)
			case nilMaybe:
				debug("    add maybeNil and visit", withType(phi))
				fs = fs.withMaybeNil(phi)
			}

			continue
		}

		for _, o1 := range operands {
			if *o1 == nil {
				continue
			}

			opNil := fs.nilness(*o1)
			if opNil == nilUnknown {
				continue
			}

			debugf("      found operand %p (%v) with nil match %v", *o1, *o1, opNil)

			if pos := instr.Pos(); pos != token.NoPos {
				if opNil == nilKnown {
					r.report(pos, "use of error variable that is known to be nil")
				} else if opNil == nilMaybe {
					r.report(pos, "use of error variable that is nil in some branches, not in others. do a nil check earlier")
				}
			} else {
				// implicit changeInterface has no position, so propagate nilness.
				if changeInterface, ok := instr.(*ssa.ChangeInterface); ok {
					fs = fs.withNilFact(changeInterface, opNil)
					debugf("    add nilFact %v for changeinterface", opNil)
					continue
				}

				debugf("      skipping unknown instruction without pos")
			}
		}
	}

	if falseBlock, nilErr, ok := isErrNilCheck(b); ok {
		// If we have an err != nil check, we can assume err == nil in falseBlock
		// only if it has no incoming edges (no other preds).
		if len(falseBlock.Preds) == 1 {
			debug("    add knownNil and visit", withType(nilErr), withType(falseBlock))
			r.visit(falseBlock, fs.withKnownNil(nilErr))
		}
	}

	// Visit all dominees of this block with the same knownNils
	for _, d := range b.Dominees() {
		r.visit(d, fs)
	}
}

// phi isn't a real reference, it's used for branch-dependent references.
// if all operands are nilKnown, then we can treat the phi value as nil.
// if some are nilKnown, then we can treat the phi value as nilMaybe.
func visitPhi(fs facts, operands []*ssa.Value) nilFact {
	var nils int
	for _, o := range operands {
		if fs.nilness(*o) == nilKnown {
			nils++
		}
	}

	// if all operands of the phi are knownNil, add result to knownNil.
	debugf("      phi instruction found %v knownNils out of %v operands", nils, len(operands))
	if len(operands) == nils {
		return nilKnown
	} else if nils > 0 {
		return nilMaybe
	}

	return nilUnknown
}

func isErrNilCheck(block *ssa.BasicBlock) (*ssa.BasicBlock, ssa.Value, bool) {
	// if instructions are always the last instruction of the containing BasicBlock.
	lastInstr := block.Instrs[len(block.Instrs)-1]

	ifInstr, ok := lastInstr.(*ssa.If)
	if !ok {
		return nil, nil, false
	}

	binCond, ok := ifInstr.Cond.(*ssa.BinOp)
	if !ok {
		return nil, nil, false
	}

	// Swap left/right operands so we can check `if err <cond> nil`.
	left, right := binCond.X, binCond.Y
	if isConstNil(left) {
		left, right = right, left
	}

	// If the comparison isn't an error against nil, don't do anything.
	if !(isErrType(left) && isConstNil(right)) {
		return nil, nil, false
	}

	trueBlock := block.Succs[0]
	falseBlock := block.Succs[1]

	switch binCond.Op {
	case token.EQL: // err == nil
		return trueBlock, left, true
	case token.NEQ: // err != nil
		return falseBlock, left, true
	default:
		return nil, nil, false
	}
}

var errType = types.Universe.Lookup("error").Type()

func isErrType(v ssa.Value) bool {
	return v.Type() == errType
}

func isConstNil(v ssa.Value) bool {
	constVal, ok := v.(*ssa.Const)
	if !ok {
		return false
	}

	return constVal.Value == nil
}

const debugEnabled = false

func debug(args ...any) {
	if !debugEnabled {
		return
	}
	fmt.Println(args...)
}

func debugf(format string, args ...any) {
	if !debugEnabled {
		return
	}
	fmt.Printf(format+"\n", args...)
}

func withType(v any) string {
	return fmt.Sprintf("%T %v", v, v)
}
