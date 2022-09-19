package server

import (
	"KeyValueStorage/storage"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShouldReturn404WhenStorageIsEmptyOnGetAllRequest(t *testing.T) {
	//when
	pairs := make([]storage.Pair[string, string], 0)
	req := httptest.NewRequest(http.MethodGet, "/pairs", nil)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("GetAll").Return(pairs)

	//then
	PairController{mockedStorage}.Get(w, req)

	//finally
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, 404, res.StatusCode)
}

func TestShouldReturnListWithPairsOnGetAllRequest(t *testing.T) {
	//when
	pairs := append(make([]storage.Pair[string, string], 0), storage.Pair[string, string]{"hop", "hey"})
	req := httptest.NewRequest(http.MethodGet, "/pairs", nil)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("GetAll").Return(pairs)

	//then
	PairController{mockedStorage}.Get(w, req)

	//finally
	res := w.Result()
	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "[{\"key\":\"hop\",\"value\":\"hey\"}]\n", string(body))
	mockedStorage.AssertCalled(t, "GetAll")
}

func TestShouldReturnPairOnGetByKeyRequest(t *testing.T) {
	//when
	pair := storage.Pair[string, string]{"hop", "hey"}
	vars := map[string]string{
		"key": pair.Key,
	}
	req := httptest.NewRequest(http.MethodGet, "/pairs/"+pair.Key, nil)
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("Get", pair.Key).Return(pair, true)

	//then
	PairController{mockedStorage}.GetByKey(w, req)

	//finally
	res := w.Result()
	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "{\"key\":\"hop\",\"value\":\"hey\"}\n", string(body))
	mockedStorage.AssertCalled(t, "Get", "hop")
}

func TestShouldReturn404OnGetByKeyWhenPairIsAbsent(t *testing.T) {
	//when
	vars := map[string]string{
		"key": "hop",
	}
	pair := storage.Pair[string, string]{}
	req := httptest.NewRequest(http.MethodGet, "/pairs/hop", nil)
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("Get", "hop").Return(pair, false)

	//then
	PairController{mockedStorage}.GetByKey(w, req)

	//finally
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, 404, res.StatusCode)
	mockedStorage.AssertCalled(t, "Get", "hop")
}

func TestShouldCreatePairOnPutRequest(t *testing.T) {
	//when
	pair := storage.Pair[string, string]{"hop", "hey"}
	pairJson, _ := json.Marshal(pair)
	body := strings.NewReader(string(pairJson))
	req := httptest.NewRequest(http.MethodGet, "/pairs/hop", body)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("AddOrUpdate", pair).Return(true)

	//then
	PairController{mockedStorage}.Put(w, req)

	//finally
	res := w.Result()
	responseBody, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	assert.Equal(t, 201, res.StatusCode)
	assert.Equal(t, "\"Successfully created!\"\n", string(responseBody))
	mockedStorage.AssertCalled(t, "AddOrUpdate", pair)
}

func TestShouldUpdatePairOnPutRequest(t *testing.T) {
	//when
	pair := storage.Pair[string, string]{"hop", "hey"}
	pairJson, _ := json.Marshal(pair)
	body := strings.NewReader(string(pairJson))
	req := httptest.NewRequest(http.MethodGet, "/pairs/hop", body)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("AddOrUpdate", pair).Return(false)

	//then
	PairController{mockedStorage}.Put(w, req)

	//finally
	res := w.Result()
	responseBody, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "\"Successfully updated!\"\n", string(responseBody))
	mockedStorage.AssertCalled(t, "AddOrUpdate", pair)
}

func TestShouldDeletePairOnDeleteRequest(t *testing.T) {
	//when
	vars := map[string]string{
		"key": "hop",
	}
	req := httptest.NewRequest(http.MethodGet, "/pairs/hop", nil)
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("Delete", "hop").Return(true)

	//then
	PairController{mockedStorage}.Delete(w, req)

	//finally
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	mockedStorage.AssertCalled(t, "Delete", "hop")
}

func TestShouldReturn404OnNonExistingPairDeletion(t *testing.T) {
	//when
	vars := map[string]string{
		"key": "hop",
	}
	req := httptest.NewRequest(http.MethodGet, "/pairs/hop", nil)
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	mockedStorage := new(storageMock[string, string])
	mockedStorage.On("Delete", "hop").Return(false)

	//then
	PairController{mockedStorage}.Delete(w, req)

	//finally
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, 404, res.StatusCode)
	mockedStorage.AssertCalled(t, "Delete", "hop")
}

type storageMock[K comparable, V comparable] struct {
	mock.Mock
}

func (s *storageMock[K, V]) AddOrUpdate(pair storage.Pair[K, V]) bool {
	args := s.Called(pair)
	return args.Get(0).(bool)
}

func (s *storageMock[K, V]) Get(key K) (storage.Pair[K, V], bool) {
	args := s.Called(key)
	return args.Get(0).(storage.Pair[K, V]), args.Get(1).(bool)
}

func (s *storageMock[K, V]) GetAll() []storage.Pair[K, V] {
	args := s.Called()
	return args.Get(0).([]storage.Pair[K, V])
}

func (s *storageMock[K, V]) Delete(key K) bool {
	args := s.Called(key)
	return args.Get(0).(bool)
}
