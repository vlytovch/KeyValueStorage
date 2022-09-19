package server

import (
	"KeyValueStorage/storage"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

// PairController handles, processes incoming requests and returns a response to the client.
// Holds a key-value storage object to manipulate inside. Keys and values are string typed.
type PairController struct {
	CurrentStorage storage.Storage[string, string]
}

// Get returns a serialized list of all pairs in the storage or 404 status in case it is empty.
func (controller PairController) Get(w http.ResponseWriter, r *http.Request) {
	values := controller.CurrentStorage.GetAll()
	if len(values) == 0 {
		http.Error(w, http.StatusText(404), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(values)
}

// GetByKey returns a serialized pair object according to key value extracted from the request URL or 404 status if the
// storage does not have key inside.
func (controller PairController) GetByKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value, exists := controller.CurrentStorage.Get(key)
	if !exists {
		http.Error(w, http.StatusText(404), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(value)
}

// Put extracts pair body from the request and add or update the storage with it. Key and value must not be nil or empty strings.
// Also notifies the client if the pair was created or updated with corresponding message.
func (controller PairController) Put(w http.ResponseWriter, r *http.Request) {
	var pair storage.Pair[string, string]
	err := json.NewDecoder(r.Body).Decode(&pair)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if pair.Key == "" || pair.Value == "" {
		http.Error(w, "Key and value should be provided!", http.StatusBadRequest)
		return
	}
	isNewPair := controller.CurrentStorage.AddOrUpdate(pair)
	if isNewPair {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode("Successfully created!")
	} else {
		json.NewEncoder(w).Encode("Successfully updated!")
	}
}

// Delete removes a pair from the storage according to key provided in request URL or 404 status if key is missing in
// the storage.
func (controller PairController) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	isDeleted := controller.CurrentStorage.Delete(key)
	if isDeleted {
		json.NewEncoder(w).Encode(key + " successfully deleted!")
		return
	}
	http.Error(w, http.StatusText(404), http.StatusNotFound)
}
