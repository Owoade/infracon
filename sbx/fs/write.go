package main

import (
	"bufio"
	"os"
)

func WriteOperation() {
	// Direct method
	jsonString := []byte("{\"name\": \"hello there\"}")
	os.WriteFile("sample.json", jsonString, 0644)

	// Using a file handler
	file, err := os.Create("sample2.json")
	check(err)

	defer file.Close()

	chunk := []byte("{\"name\": \"hello there again\"}")

	file.Write(chunk)

	file.Sync()

	// Using buffer writer
	secondFile, err := os.Create("sample3.json")
	check(err)

	writter := bufio.NewWriter(secondFile)
	writter.Write([]byte("{\"name\": \"hello there again for the last time\"}"))
	
	writter.Flush()
}
