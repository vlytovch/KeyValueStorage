package mapStorage

import (
	"KeyValueStorage/storage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldGetEmptyListOnGettingElementsFromNewStorage(t *testing.T) {
	mapStorage := NewStorage[string, string]()
	list := mapStorage.GetAll()
	assert.Empty(t, list)
}

func TestShouldGetSomeElementsAfterAdding(t *testing.T) {
	mapStorage := NewStorage[string, string]()
	pairToCreate := storage.Pair[string, string]{"burak-crush-key", "burak-crush-value"}
	pairToUpdate := storage.Pair[string, string]{"burak-crush-update-key", "burak-crush-update-value"}

	mapStorage.AddOrUpdate(pairToCreate)
	mapStorage.AddOrUpdate(pairToUpdate)
	assert.Equal(t, len(mapStorage.GetAll()), 2)
}

func TestShouldCheckIfExistingPairWithKeyGetsUpdated(t *testing.T) {
	mapStorage := NewStorage[string, string]()
	pairToCreate := storage.Pair[string, string]{"burak-crush-key", "burak-crush-value"}
	pairToUpdate := storage.Pair[string, string]{"burak-crush-key", "burak-crush-update-value"}

	isCreated := mapStorage.AddOrUpdate(pairToCreate)
	assert.True(t, isCreated)

	isCreated = mapStorage.AddOrUpdate(pairToUpdate)
	assert.False(t, isCreated)
	assert.Equal(t, len(mapStorage.GetAll()), 1)
}

func TestShouldGetAddedToStoragePairByKey(t *testing.T) {
	expectedPair := storage.Pair[string, string]{"burak-crush-key", "burak-crush-value"}
	mapStorage := NewStorage[string, string]()

	_, isPresent := mapStorage.Get("burak-crush-key")
	assert.False(t, isPresent)

	mapStorage.AddOrUpdate(expectedPair)
	value, isPresentAfterAdd := mapStorage.Get("burak-crush-key")
	assert.True(t, isPresentAfterAdd)
	assert.Equal(t, value, expectedPair)
}

func TestShouldCheckIfExistingInStoragePairGetsDeleted(t *testing.T) {
	expectedPair := storage.Pair[string, string]{"burak-crush-key", "burak-crush-value"}
	mapStorage := NewStorage[string, string]()

	isDeleted := mapStorage.Delete("burak-crush-key")
	assert.False(t, isDeleted)
	mapStorage.AddOrUpdate(expectedPair)
	_, isPresent := mapStorage.Get("burak-crush-key")
	assert.True(t, isPresent)

	isDeleted = mapStorage.Delete("burak-crush-key")
	assert.True(t, isDeleted)
	_, isPresent = mapStorage.Get("burak-crush-key")
	assert.False(t, isPresent)
}
