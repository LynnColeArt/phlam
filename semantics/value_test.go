package semantics

import (
	"math"
	"math/big"
	"testing"
)

func FuzzCheckedArithmetic(f *testing.F) {
	for _, pair := range [][2]int64{{0, 0}, {1, -1}, {math.MaxInt64, 1}, {math.MinInt64, -1}, {math.MinInt64, 1}, {math.MaxInt64, math.MaxInt64}, {-3, -7}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, a, b int64) {
		for _, op := range []struct {
			name   string
			exact  func(int64, int64) (int64, error)
			oracle func(*big.Int, *big.Int, *big.Int) *big.Int
		}{
			{"add", Add, (*big.Int).Add}, {"sub", Sub, (*big.Int).Sub}, {"mul", Mul, (*big.Int).Mul},
		} {
			want := op.oracle(new(big.Int), big.NewInt(a), big.NewInt(b))
			got, err := op.exact(a, b)
			if want.IsInt64() {
				if err != nil || got != want.Int64() {
					t.Fatalf("%s(%d,%d): %d %v; want %s", op.name, a, b, got, err, want)
				}
			} else if err != ErrOverflow {
				t.Fatalf("missing overflow: %s(%d,%d)", op.name, a, b)
			}
		}
		got, err := Neg(a)
		if a == math.MinInt64 {
			if err != ErrOverflow {
				t.Fatal("missing neg overflow")
			}
		} else if err != nil || got != -a {
			t.Fatal("wrong negation")
		}
	})
}
