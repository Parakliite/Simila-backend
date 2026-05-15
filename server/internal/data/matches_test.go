package data

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := mat.NewVecDense(3, []float64{1, 2, 3})
	b := mat.NewVecDense(3, []float64{1, 2, 3})

	got := cosineSimilarity(a, b)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("expected 1.0, got %f", got)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := mat.NewVecDense(3, []float64{1, 0, 0})
	b := mat.NewVecDense(3, []float64{0, 1, 0})

	got := cosineSimilarity(a, b)
	if math.Abs(got) > 1e-9 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := mat.NewVecDense(3, []float64{1, 2, 3})
	b := mat.NewVecDense(3, []float64{-1, -2, -3})

	got := cosineSimilarity(a, b)
	if math.Abs(got-(-1.0)) > 1e-9 {
		t.Errorf("expected -1.0, got %f", got)
	}
}

func TestCosineSimilarity_OneZeroVector(t *testing.T) {
	a := mat.NewVecDense(3, []float64{1, 2, 3})
	b := mat.NewVecDense(3, []float64{0, 0, 0})

	got := cosineSimilarity(a, b)
	if got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestCosineSimilarity_BothZeroVectors(t *testing.T) {
	a := mat.NewVecDense(3, []float64{0, 0, 0})
	b := mat.NewVecDense(3, []float64{0, 0, 0})

	got := cosineSimilarity(a, b)
	if got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestCosineSimilarity_PartialOverlap(t *testing.T) {
	a := mat.NewVecDense(4, []float64{5, 0, 3, 0})
	b := mat.NewVecDense(4, []float64{5, 0, 0, 4})

	got := cosineSimilarity(a, b)
	if got <= 0 || got >= 1.0 {
		t.Errorf("expected similarity between 0 and 1, got %f", got)
	}
}
