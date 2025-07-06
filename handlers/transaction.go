package handlers

import (
	"fmt"
	"net/http"
)

func Transaction(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Hello world")
}
