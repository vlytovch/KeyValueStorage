package main

import (
	"KeyValueStorage/mapStorage"
	"KeyValueStorage/server"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	var newStorage = mapStorage.NewStorage[string, string]()
	var controller = server.PairController{CurrentStorage: newStorage}

	router := mux.NewRouter()
	router.HandleFunc("/pairs", controller.Get).Methods("GET")
	router.HandleFunc("/pairs", controller.Put).Methods("PUT")
	router.HandleFunc("/pairs/{key:.+}", controller.GetByKey).Methods("GET")
	router.HandleFunc("/pairs/{key:.+}", controller.Delete).Methods("DELETE")
	http.Handle("/", router)

	fmt.Println("Server is listening...")
	http.ListenAndServe(":8181", nil)
}
