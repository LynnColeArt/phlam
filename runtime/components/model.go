package components

import (
	"fmt"
	"math"
	"math/rand"
)

const Scale = 32.0
const Epochs = 12000
const Rate = 0.3

type Model struct {
	Weights [][]float64 `json:"weights"`
	Hidden  [][]float64 `json:"hidden,omitempty"`
}
type Sample struct {
	Operation  int        `json:"operation_index"`
	State      int64      `json:"state"`
	Arg        int64      `json:"argument"`
	Prediction Prediction `json:"prediction"`
	TraceHash  string     `json:"observed_trace_sha256"`
}

func features(count, index int, state, arg int64) []float64 {
	x := make([]float64, 3+count)
	x[0] = 1
	x[1] = float64(state) / Scale
	x[2] = float64(arg) / Scale
	x[3+index] = 1
	return x
}
func (m Model) Predict(count, index int, state, arg int64) (out [3]float64) {
	x := features(count, index, state, arg)
	base := len(x)
	for _, row := range m.Hidden {
		v := 0.0
		for j := 0; j < base; j++ {
			v += row[j] * x[j]
		}
		x = append(x, math.Max(0, v))
	}
	for h := range out {
		for j, v := range x {
			out[h] += m.Weights[h][j] * v
		}
	}
	return out
}
func Decode(scores [3]float64) (Prediction, error) {
	for _, v := range scores {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 8 {
			return Prediction{}, fmt.Errorf("invalid numerical prediction")
		}
	}
	return Prediction{Next: int64(math.Round(scores[0] * Scale)), Result: int64(math.Round(scores[1] * Scale)), Write: scores[2] >= .5}, nil
}
func Train(samples []Sample, count int, seed int64) (Model, float64, float64, error) {
	if len(samples) == 0 || count < 1 || count > 8 {
		return Model{}, 0, 0, fmt.Errorf("invalid training task")
	}
	random := rand.New(rand.NewSource(seed))
	m := Model{Weights: make([][]float64, 3)}
	for h := range m.Weights {
		m.Weights[h] = make([]float64, 3+count)
		for j := range m.Weights[h] {
			m.Weights[h][j] = (random.Float64() - .5) * .1
		}
	}
	inputs := make([][]float64, len(samples))
	targets := make([][3]float64, len(samples))
	for i, s := range samples {
		if s.Operation < 0 || s.Operation >= count {
			return Model{}, 0, 0, fmt.Errorf("bad training operation")
		}
		inputs[i] = features(count, s.Operation, s.State, s.Arg)
		write := 0.0
		if s.Prediction.Write {
			write = 1
		}
		targets[i] = [3]float64{float64(s.Prediction.Next) / Scale, float64(s.Prediction.Result) / Scale, write}
	}
	loss := func() float64 {
		sum := 0.0
		for i, s := range samples {
			p := m.Predict(count, s.Operation, s.State, s.Arg)
			for h := range p {
				d := p[h] - targets[i][h]
				sum += d * d
			}
		}
		return sum / float64(3*len(samples))
	}
	initial := loss()
	for epoch := 0; epoch < Epochs; epoch++ {
		gradient := make([][]float64, 3)
		for h := range gradient {
			gradient[h] = make([]float64, 3+count)
		}
		for i, x := range inputs {
			for h := range gradient {
				prediction := 0.0
				for j, v := range x {
					prediction += m.Weights[h][j] * v
				}
				d := prediction - targets[i][h]
				for j, v := range x {
					gradient[h][j] += d * v
				}
			}
		}
		for h := range gradient {
			for j := range gradient[h] {
				m.Weights[h][j] -= 2 * Rate * gradient[h][j] / float64(len(samples))
			}
		}
	}
	return m, initial, loss(), nil
}
