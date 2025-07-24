package vector

import (
	"math"
)

// CosineSimilarity calculates cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical vectors
func CosineSimilarity(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0.0
	}
	
	if len(a) == 0 {
		return 0.0
	}
	
	var dotProduct, normA, normB float32
	
	// Calculate dot product and norms in single pass
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	// Avoid division by zero
	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistance calculates Euclidean distance between two vectors
// Lower values indicate higher similarity
func EuclideanDistance(a, b Vector) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}
	
	if len(a) == 0 {
		return 0.0
	}
	
	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	
	return float32(math.Sqrt(float64(sum)))
}

// DotProduct calculates dot product between two vectors
func DotProduct(a, b Vector) float32 {
	if len(a) != len(b) {
		return 0.0
	}
	
	var product float32
	for i := 0; i < len(a); i++ {
		product += a[i] * b[i]
	}
	
	return product
}

// Magnitude calculates the magnitude (L2 norm) of a vector
func Magnitude(v Vector) float32 {
	var sum float32
	for _, val := range v {
		sum += val * val
	}
	return float32(math.Sqrt(float64(sum)))
}

// Normalize normalizes a vector to unit length
// Returns a new vector; does not modify the original
func Normalize(v Vector) Vector {
	mag := Magnitude(v)
	if mag == 0.0 {
		return make(Vector, len(v)) // return zero vector
	}
	
	normalized := make(Vector, len(v))
	for i, val := range v {
		normalized[i] = val / mag
	}
	
	return normalized
}

// BatchCosineSimilarity calculates cosine similarity between a query vector
// and multiple vectors in a batch, returning similarity scores
func BatchCosineSimilarity(query Vector, vectors []Vector) []float32 {
	if len(vectors) == 0 {
		return nil
	}
	
	similarities := make([]float32, len(vectors))
	
	// Precompute query norm
	var queryNorm float32
	for _, val := range query {
		queryNorm += val * val
	}
	queryNorm = float32(math.Sqrt(float64(queryNorm)))
	
	if queryNorm == 0.0 {
		return similarities // all zeros
	}
	
	// Calculate similarities
	for i, vec := range vectors {
		if len(vec) != len(query) {
			similarities[i] = 0.0
			continue
		}
		
		var dotProduct, vecNorm float32
		for j := 0; j < len(query); j++ {
			dotProduct += query[j] * vec[j]
			vecNorm += vec[j] * vec[j]
		}
		
		vecNorm = float32(math.Sqrt(float64(vecNorm)))
		if vecNorm == 0.0 {
			similarities[i] = 0.0
		} else {
			similarities[i] = dotProduct / (queryNorm * vecNorm)
		}
	}
	
	return similarities
}

// TopKIndices returns the indices of the top-k highest values in scores
func TopKIndices(scores []float32, k int) []int {
	if k <= 0 || len(scores) == 0 {
		return nil
	}
	
	if k >= len(scores) {
		k = len(scores)
	}
	
	// Create index-score pairs
	type indexScore struct {
		index int
		score float32
	}
	
	pairs := make([]indexScore, len(scores))
	for i, score := range scores {
		pairs[i] = indexScore{index: i, score: score}
	}
	
	// Partial sort to get top-k elements
	// Using simple selection sort for small k, heap sort would be better for large k
	for i := 0; i < k; i++ {
		maxIdx := i
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].score > pairs[maxIdx].score {
				maxIdx = j
			}
		}
		pairs[i], pairs[maxIdx] = pairs[maxIdx], pairs[i]
	}
	
	// Extract indices
	indices := make([]int, k)
	for i := 0; i < k; i++ {
		indices[i] = pairs[i].index
	}
	
	return indices
}