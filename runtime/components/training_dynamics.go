package components

import "math"

type UnitDynamics struct {
	Active      int         `json:"active_samples"`
	ByOperation map[int]int `json:"active_by_operation"`
	Min         float64     `json:"min_preactivation"`
	Max         float64     `json:"max_preactivation"`
	GradientL2  float64     `json:"current_hidden_gradient_l2"`
}
type TrainingSnapshot struct {
	Epoch            int            `json:"completed_epochs"`
	MSE              float64        `json:"training_mse"`
	Model            Model          `json:"model"`
	Units            []UnitDynamics `json:"units"`
	OutputGradientL2 float64        `json:"current_output_gradient_l2"`
}

func trainingCheckpoint(epoch int) bool {
	switch epoch {
	case 0, 1, 10, 100, 500, 1000, 3000, 6000, 12000, 18000:
		return true
	}
	return false
}
func trainingSnapshot(m Model, xs [][]float64, gradient [][]float64, width, epoch int, loss float64) TrainingSnapshot {
	clone := Model{}
	for _, row := range m.Hidden {
		clone.Hidden = append(clone.Hidden, append([]float64(nil), row...))
	}
	for _, row := range m.Weights {
		clone.Weights = append(clone.Weights, append([]float64(nil), row...))
	}
	out := TrainingSnapshot{Epoch: epoch, MSE: loss, Model: clone}
	for k, row := range m.Hidden {
		u := UnitDynamics{Min: math.Inf(1), Max: math.Inf(-1), ByOperation: map[int]int{}}
		for _, x := range xs {
			v := 0.0
			operation := 0
			for j, z := range x {
				v += row[j] * z
				if j >= 3 && z == 1 {
					operation = j - 3
				}
			}
			u.Min = math.Min(u.Min, v)
			u.Max = math.Max(u.Max, v)
			if v > 0 {
				u.Active++
				u.ByOperation[operation]++
			}
		}
		for _, g := range gradient[k] {
			u.GradientL2 += g * g
		}
		u.GradientL2 = math.Sqrt(u.GradientL2)
		out.Units = append(out.Units, u)
	}
	for _, row := range gradient[width:] {
		for _, g := range row {
			out.OutputGradientL2 += g * g
		}
	}
	out.OutputGradientL2 = math.Sqrt(out.OutputGradientL2)
	return out
}
