package server

import "net/http"

func main() {

	err := http.ListenAndServe(":2000", nil)

	if err != nil {
		panic(err)
	}
	

}
