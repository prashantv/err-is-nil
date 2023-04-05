package errisnil

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrIsNil(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, ".")
}
