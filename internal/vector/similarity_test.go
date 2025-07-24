package vector

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        Vector
		b        Vector
		expected float32
		epsilon  float32
	}{
		{
			name:     "identical vectors",
			a:        Vector{1, 2, 3},
			b:        Vector{1, 2, 3},
			expected: 1.0,
			epsilon:  1e-6,
		},
		{
			name:     "orthogonal vectors",
			a:        Vector{1, 0, 0},
			b:        Vector{0, 1, 0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "opposite vectors",
			a:        Vector{1, 0, 0},
			b:        Vector{-1, 0, 0},
			expected: -1.0,
			epsilon:  1e-6,
		},
		{
			name:     "empty vectors",
			a:        Vector{},
			b:        Vector{},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "different dimensions",
			a:        Vector{1, 2},
			b:        Vector{1, 2, 3},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "zero vector",
			a:        Vector{0, 0, 0},
			b:        Vector{1, 2, 3},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "normalized vectors",
			a:        Vector{0.6, 0.8},
			b:        Vector{0.8, 0.6},
			expected: 0.96,
			epsilon:  1e-5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > float64(tt.epsilon) {
				t.Errorf("CosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        Vector
		b        Vector
		expected float32
		epsilon  float32
	}{
		{
			name:     "identical vectors",
			a:        Vector{1, 2, 3},
			b:        Vector{1, 2, 3},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "unit distance",
			a:        Vector{0, 0, 0},
			b:        Vector{1, 0, 0},
			expected: 1.0,
			epsilon:  1e-6,
		},
		{
			name:     "diagonal distance",
			a:        Vector{0, 0},
			b:        Vector{3, 4},
			expected: 5.0,
			epsilon:  1e-6,
		},
		{
			name:     "empty vectors",
			a:        Vector{},
			b:        Vector{},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "different dimensions",
			a:        Vector{1, 2},
			b:        Vector{1, 2, 3},
			expected: float32(math.Inf(1)),
			epsilon:  0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EuclideanDistance(tt.a, tt.b)
			if math.IsInf(float64(tt.expected), 1) {
				if !math.IsInf(float64(result), 1) {
					t.Errorf("EuclideanDistance(%v, %v) = %v, want +Inf", tt.a, tt.b, result)
				}
			} else if math.Abs(float64(result-tt.expected)) > float64(tt.epsilon) {
				t.Errorf("EuclideanDistance(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        Vector
		b        Vector
		expected float32
	}{
		{
			name:     "simple dot product",
			a:        Vector{1, 2, 3},
			b:        Vector{4, 5, 6},
			expected: 32, // 1*4 + 2*5 + 3*6
		},
		{
			name:     "orthogonal vectors",
			a:        Vector{1, 0},
			b:        Vector{0, 1},
			expected: 0,
		},
		{
			name:     "empty vectors",
			a:        Vector{},
			b:        Vector{},
			expected: 0,
		},
		{
			name:     "different dimensions",
			a:        Vector{1, 2},
			b:        Vector{1, 2, 3},
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DotProduct(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("DotProduct(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMagnitude(t *testing.T) {
	tests := []struct {
		name     string
		vector   Vector
		expected float32
		epsilon  float32
	}{
		{
			name:     "unit vector",
			vector:   Vector{1, 0, 0},
			expected: 1.0,
			epsilon:  1e-6,
		},
		{
			name:     "3-4-5 triangle",
			vector:   Vector{3, 4},
			expected: 5.0,
			epsilon:  1e-6,
		},
		{
			name:     "zero vector",
			vector:   Vector{0, 0, 0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "empty vector",
			vector:   Vector{},
			expected: 0.0,
			epsilon:  1e-6,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Magnitude(tt.vector)
			if math.Abs(float64(result-tt.expected)) > float64(tt.epsilon) {
				t.Errorf("Magnitude(%v) = %v, want %v", tt.vector, result, tt.expected)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name         string
		vector       Vector
		expectedMag  float32
		epsilon      float32
	}{
		{
			name:        "simple vector",
			vector:      Vector{3, 4},
			expectedMag: 1.0,
			epsilon:     1e-6,
		},
		{
			name:        "zero vector",
			vector:      Vector{0, 0, 0},
			expectedMag: 0.0,
			epsilon:     1e-6,
		},
		{
			name:        "already normalized",
			vector:      Vector{1, 0, 0},
			expectedMag: 1.0,
			epsilon:     1e-6,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.vector)
			mag := Magnitude(result)
			if math.Abs(float64(mag-tt.expectedMag)) > float64(tt.epsilon) {
				t.Errorf("Normalize(%v) magnitude = %v, want %v", tt.vector, mag, tt.expectedMag)
			}
			
			// Ensure original vector is not modified
			if len(tt.vector) > 0 && len(result) > 0 {
				originalMag := Magnitude(tt.vector)
				if tt.vector[0] != 0 && result[0] == tt.vector[0] && originalMag != 1.0 {
					t.Error("Normalize modified the original vector")
				}
			}
		})
	}
}

func TestBatchCosineSimilarity(t *testing.T) {
	query := Vector{1, 0, 0}
	vectors := []Vector{
		{1, 0, 0}, // identical
		{0, 1, 0}, // orthogonal
		{-1, 0, 0}, // opposite
		{0.5, 0, 0}, // same direction, different magnitude
	}
	
	expected := []float32{1.0, 0.0, -1.0, 1.0}
	
	results := BatchCosineSimilarity(query, vectors)
	
	if len(results) != len(expected) {
		t.Fatalf("BatchCosineSimilarity returned %d results, want %d", len(results), len(expected))
	}
	
	for i, result := range results {
		if math.Abs(float64(result-expected[i])) > 1e-6 {
			t.Errorf("BatchCosineSimilarity result[%d] = %v, want %v", i, result, expected[i])
		}
	}
}

func TestTopKIndices(t *testing.T) {
	tests := []struct {
		name     string
		scores   []float32
		k        int
		expected []int
	}{
		{
			name:     "simple top-k",
			scores:   []float32{0.1, 0.9, 0.3, 0.7, 0.5},
			k:        3,
			expected: []int{1, 3, 4}, // indices of scores 0.9, 0.7, 0.5
		},
		{
			name:     "k larger than length",
			scores:   []float32{0.1, 0.2},
			k:        5,
			expected: []int{1, 0},
		},
		{
			name:     "k is zero",
			scores:   []float32{0.1, 0.2, 0.3},
			k:        0,
			expected: nil,
		},
		{
			name:     "empty scores",
			scores:   []float32{},
			k:        3,
			expected: nil,
		},
		{
			name:     "duplicate scores",
			scores:   []float32{0.5, 0.5, 0.3, 0.5},
			k:        2,
			expected: []int{0, 1}, // any two indices with score 0.5
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TopKIndices(tt.scores, tt.k)
			
			if len(result) != len(tt.expected) {
				t.Errorf("TopKIndices returned %d indices, want %d", len(result), len(tt.expected))
				return
			}
			
			if tt.expected == nil {
				if result != nil {
					t.Errorf("TopKIndices returned %v, want nil", result)
				}
				return
			}
			
			// Verify that returned indices are valid and scores are in descending order
			for i := 0; i < len(result)-1; i++ {
				if result[i] < 0 || result[i] >= len(tt.scores) {
					t.Errorf("Invalid index %d returned", result[i])
				}
				if tt.scores[result[i]] < tt.scores[result[i+1]] {
					t.Errorf("Scores not in descending order: %v at index %d < %v at index %d", 
						tt.scores[result[i]], result[i], tt.scores[result[i+1]], result[i+1])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkCosineSimilarity(b *testing.B) {
	vec1 := make(Vector, 1536) // OpenAI embedding dimension
	vec2 := make(Vector, 1536)
	
	// Initialize with some values
	for i := range vec1 {
		vec1[i] = float32(i % 100) / 100.0
		vec2[i] = float32((i+50) % 100) / 100.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(vec1, vec2)
	}
}

func BenchmarkBatchCosineSimilarity(b *testing.B) {
	query := make(Vector, 1536)
	vectors := make([]Vector, 1000)
	
	// Initialize vectors
	for i := range query {
		query[i] = float32(i % 100) / 100.0
	}
	
	for i := range vectors {
		vectors[i] = make(Vector, 1536)
		for j := range vectors[i] {
			vectors[i][j] = float32((j+i) % 100) / 100.0
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchCosineSimilarity(query, vectors)
	}
}