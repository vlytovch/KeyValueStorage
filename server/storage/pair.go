package storage

// Pair holds a key and value to be persisted in the Storage.
// Use generics so and type that is comparable can be applied for key and value.
type Pair[K comparable, V comparable] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}
