// Package mapStorage is an implementation of key value storage that persists the data in a map
package mapStorage

import "KeyValueStorage/storage"

// MapStorage holds a map object for persisting key-value pairs inside.
// It is supplied with generics so any comparable type can be provided for key and value.
type MapStorage[K comparable, V comparable] struct {
	data map[K]V
}

// NewStorage generates a new MapStorage with initialized empty map inside and provided generic types for keys and values.
func NewStorage[K comparable, V comparable]() *MapStorage[K, V] {
	return &MapStorage[K, V]{make(map[K]V)}
}

// AddOrUpdate complements the map with a new pair or update existing one if the pair with such key already exists.
// Returns true if the pair was added and false if it was updated.
func (mapStorage *MapStorage[K, V]) AddOrUpdate(pair storage.Pair[K, V]) bool {
	_, isNewPair := mapStorage.data[pair.Key]
	mapStorage.data[pair.Key] = pair.Value
	return !isNewPair
}

// Get returns a pair and boolean indicating if value exists in the map according to provided key.
// An empty pair object will be returned in case there is no such key in the map yet.
func (mapStorage *MapStorage[K, V]) Get(key K) (storage.Pair[K, V], bool) {
	value, exists := mapStorage.data[key]
	if exists {
		return storage.Pair[K, V]{key, value}, exists
	}
	return storage.Pair[K, V]{}, exists
}

// GetAll returns a list of all pairs persisted in the map.
func (mapStorage *MapStorage[K, V]) GetAll() []storage.Pair[K, V] {
	list := make([]storage.Pair[K, V], 0, len(mapStorage.data))
	for key, value := range mapStorage.data {
		list = append(list, storage.Pair[K, V]{key, value})
	}
	return list
}

// Delete removes a pair from the map in case the key exists there.
// Returns true if the pair has been deleted or false if provided key is missing in the map.
func (mapStorage *MapStorage[K, V]) Delete(key K) bool {
	_, isPresent := mapStorage.Get(key)
	if isPresent {
		delete(mapStorage.data, key)
		return true
	}
	return false
}
