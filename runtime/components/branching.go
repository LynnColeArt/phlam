package components

import (
	"fmt"
	"math"
	"math/rand"
)

const BranchEpochs = 24000
const BranchWidth = 16

// TrainBranching fits a residual one-hidden-layer ReLU network with full-batch
// Adam. Only sample inputs and targets reach the optimizer: no contract paths,
// thresholds, certification observations, or peer model enter its features.
func TrainBranching(samples []Sample, count int, seed int64) (Model, float64, float64, error) {
	return TrainBranchingWidth(samples, count, seed, BranchWidth)
}

// TrainBranchingWidth uses the fixed optimizer and budget with an explicitly
// selected hidden width. Width selection never changes input features or targets.
func TrainBranchingWidth(samples []Sample, count int, seed int64, width int) (Model, float64, float64, error) {
	return TrainBranchingObserved(samples, count, seed, width, "legacy", nil)
}

// TrainBranchingObserved adds passive training-only snapshots. It receives no
// contracts, oracle-domain observations or certification feedback.
func TrainBranchingObserved(samples []Sample, count int, seed int64, width int, initialization string, observe func(TrainingSnapshot)) (Model, float64, float64, error) {
	if len(samples) == 0 || count < 1 || count > 8 || width < 1 || width > 32 {
		return Model{}, 0, 0, fmt.Errorf("invalid training task")
	}
	if initialization != "legacy" && initialization != "paired-centered" && initialization != "paired-affine" {
		return Model{}, 0, 0, fmt.Errorf("unknown initialization")
	}
	base := 3 + count
	random := rand.New(rand.NewSource(seed))
	m := Model{Hidden: make([][]float64, width), Weights: make([][]float64, 3)}
	for h := range m.Hidden {
		m.Hidden[h] = make([]float64, base)
		for j := range m.Hidden[h] {
			m.Hidden[h][j] = (random.Float64() - .5) * 1.0
		}
	}
	for h := range m.Weights {
		m.Weights[h] = make([]float64, base+width)
		for j := range m.Weights[h] {
			m.Weights[h][j] = (random.Float64() - .5) * .2
		}
	}
	if initialization != "legacy" {
		// Center every random hyperplane and pair opposing directions. No target,
		// source threshold, contract or finite-domain bound enters this geometry.
		for k, row := range m.Hidden {
			row[0] = 0
			for j := 3; j < base; j++ {
				row[j] = 0
			}
			if k%2 == 1 {
				row[1] = -m.Hidden[k-1][1]
				row[2] = -m.Hidden[k-1][2]
			}
		}
	}
	if initialization == "paired-affine" {
		// A distinct, explicitly accounted 12,000-epoch affine prefit uses only
		// these same training examples. Hidden-output coefficients start at zero;
		// all layers remain trainable during the unchanged joint Adam phase.
		linear, _, _, err := Train(samples, count, seed)
		if err != nil {
			return Model{}, 0, 0, err
		}
		for h, row := range m.Weights {
			clear(row)
			copy(row, linear.Weights[h])
		}
	}
	xs := make([][]float64, len(samples))
	ys := make([][3]float64, len(samples))
	for i, s := range samples {
		if s.Operation < 0 || s.Operation >= count {
			return Model{}, 0, 0, fmt.Errorf("bad training operation")
		}
		xs[i] = features(count, s.Operation, s.State, s.Arg)
		w := 0.0
		if s.Prediction.Write {
			w = 1
		}
		ys[i] = [3]float64{float64(s.Prediction.Next) / Scale, float64(s.Prediction.Result) / Scale, w}
	}
	loss := func() float64 {
		sum := 0.0
		for i, s := range samples {
			p := m.Predict(count, s.Operation, s.State, s.Arg)
			for h := range p {
				d := p[h] - ys[i][h]
				sum += d * d
			}
		}
		return sum / float64(3*len(samples))
	}
	initial := loss()
	rows := append(append([][]float64{}, m.Hidden...), m.Weights...)
	grad := make([][]float64, len(rows))
	first := make([][]float64, len(rows))
	second := make([][]float64, len(rows))
	for i, row := range rows {
		grad[i] = make([]float64, len(row))
		first[i] = make([]float64, len(row))
		second[i] = make([]float64, len(row))
	}
	activ := make([]float64, base+width)
	gradient := func() {
		for _, row := range grad {
			clear(row)
		}
		for i, x := range xs {
			copy(activ, x)
			for k, row := range m.Hidden {
				v := 0.0
				for j, z := range x {
					v += row[j] * z
				}
				activ[base+k] = math.Max(0, v)
			}
			var delta [3]float64
			for h, row := range m.Weights {
				v := 0.0
				for j, z := range activ {
					v += row[j] * z
				}
				delta[h] = 2 * (v - ys[i][h]) / float64(3*len(samples))
				for j, z := range activ {
					grad[width+h][j] += delta[h] * z
				}
			}
			for k := range m.Hidden {
				if activ[base+k] <= 0 {
					continue
				}
				d := 0.0
				for h := range delta {
					d += delta[h] * m.Weights[h][base+k]
				}
				for j, z := range x {
					grad[k][j] += d * z
				}
			}
		}
	}
	for epoch := 1; epoch <= BranchEpochs; epoch++ {
		gradient()
		if observe != nil && trainingCheckpoint(epoch-1) {
			observe(trainingSnapshot(m, xs, grad, width, epoch-1, loss()))
		}
		// A fixed annealing schedule; no held-out or certificate feedback.
		rate := .003
		if epoch > BranchEpochs*3/4 {
			rate = .0003
		}
		b1, b2 := 1-math.Pow(.9, float64(epoch)), 1-math.Pow(.999, float64(epoch))
		for i, row := range rows {
			for j := range row {
				g := grad[i][j]
				first[i][j] = .9*first[i][j] + .1*g
				second[i][j] = .999*second[i][j] + .001*g*g
				row[j] -= rate * (first[i][j] / b1) / (math.Sqrt(second[i][j]/b2) + 1e-8)
			}
		}
	}
	if observe != nil {
		gradient()
		observe(trainingSnapshot(m, xs, grad, width, BranchEpochs, loss()))
	}
	return m, initial, loss(), nil
}
