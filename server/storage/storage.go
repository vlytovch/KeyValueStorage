package storage

// Storage describes a way to access some persistent data.
type Storage[K comparable, V comparable] interface {
	AddOrUpdate(pair Pair[K, V]) bool
	Get(key K) (Pair[K, V], bool)
	GetAll() []Pair[K, V]
	Delete(key K) bool
}
