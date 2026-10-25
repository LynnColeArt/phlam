package components

import (
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"os"
	"strings"
	"testing"
)

func TestGuardedContractAndFrame(t *testing.T) {
	source, err := os.ReadFile("../../examples/accumulator/accumulator.phlam")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Compile(string(source), "Accumulator")
	if err != nil {
		t.Fatal(err)
	}
	if contract.Slot != "total" || len(contract.Operations) != 2 || len(contract.Operations[0].Paths) != 3 {
		t.Fatal("guarded paths not derived", contract)
	}
	for state := int64(-20); state <= 20; state++ {
		for arg := int64(-12); arg <= 12; arg++ {
			var work abi.Work
			expected, err := contract.Operations[0].Evaluate(state, arg, &work)
			if err != nil {
				t.Fatal(err)
			}
			accepted := arg >= -8 && arg <= 8 && state+arg >= -16 && state+arg <= 16
			if expected.Accepted != accepted || accepted && (expected.Prediction.Next != state+arg || expected.Prediction.Result != state+arg || !expected.Prediction.Write) {
				t.Fatal("wrong guard relation", state, arg, expected)
			}
		}
	}
	sourceText := strings.Replace(string(source), "$this->total = $next", "$other->total = $next", 1)
	if _, err := Compile(sourceText, "Accumulator"); err == nil {
		t.Fatal("cross-instance write accepted")
	}
}
func TestRestrictedCompilerRejectsUnsupportedShapes(t *testing.T) {
	for _, source := range []string{
		"/// Missing init.\nclass Missing {$value=0; function read(){return $this->value}}",
		"/// Multiple slots.\nclass Multi {$x=0; $y=0; function init($n){$this->x=$n} function read(){return $this->x}}",
		"/// Calls.\nclass Calls {$x=0; function init($n){$this->x=$n} function run(){return eval('return 1')}}",
		"/// Missing return.\nclass NoReturn {$x=0; function init($n){$this->x=$n} function run(){$this->x=1}}",
	} {
		name := strings.Split(strings.Split(source, "class ")[1], " ")[0]
		if _, err := Compile(source, name); err == nil {
			t.Fatal("unsupported component accepted", name)
		}
	}
}

func TestSymbolicGrowthIsBounded(t *testing.T) {
	source := "/// Bounded analysis.\nclass Growth {$value=0; function init($start){$this->value=$start} function run(){$x=$this->value;" + strings.Repeat("$x=$x+$x;", 20) + "return $x}}"
	if _, err := Compile(source, "Growth"); err == nil {
		t.Fatal("unbounded symbolic expansion accepted")
	}
}

func TestSaturatingPathsAgainstMathematicalSpecification(t *testing.T) {
	source, err := os.ReadFile("../../examples/saturating-accumulator/saturating-accumulator.phlam")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Compile(string(source), "SaturatingAccumulator")
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.Operations[0].Paths) != 3 {
		t.Fatal("expected lower, upper and interior paths")
	}
	for state := int64(-20); state <= 20; state++ {
		for arg := int64(-12); arg <= 12; arg++ {
			want := state + arg
			if want < -16 {
				want = -16
			}
			if want > 16 {
				want = 16
			}
			var work abi.Work
			got, err := contract.Operations[0].Evaluate(state, arg, &work)
			if err != nil || !got.Accepted || got.Prediction != (abi.Prediction{Next: want, Result: want, Write: true}) {
				t.Fatal(state, arg, got, err)
			}
		}
	}
}
