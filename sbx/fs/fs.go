package main

func main() {
	ReadOperation()
	WriteOperation()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
