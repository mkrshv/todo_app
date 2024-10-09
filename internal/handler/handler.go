package handler

import (
	"fmt"
	"net/http"
)

func StartServer(port string) {
	webDir := "./web"

	http.Handle("/", http.FileServer(http.Dir(webDir)))

	fmt.Println("Server starting at", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
