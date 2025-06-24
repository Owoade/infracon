package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func ReadOperation() {
	// Reading all the data
	data, err := os.ReadFile("test.json")
	check(err)
	fmt.Println(string(data), data)

	f, err := os.Open("test.json")
	check(err)

	// Reading a chunk of a file
	fileSlice := make([]byte, 5)
	chunk, err := f.Read(fileSlice)
	check(err)

	fmt.Println(fileSlice[:chunk])

	//Moving a file pointer
	offset, err := f.Seek(10, io.SeekStart)
	check(err)

	fmt.Println(offset)

	fileSlice2 := make([]byte, 5)
	chunk2, err := f.Read(fileSlice2)
	fmt.Println(string(fileSlice2[:chunk2]))

	fileSlice3 := make([]byte, 5)
	chunk3, _ := io.ReadAtLeast(f, fileSlice3, 3)
	fmt.Println(string(fileSlice3[:chunk3]))

	// Reading chunks from a file without creating a slice
	reader := bufio.NewReader(f)
	data2, _ := reader.Peek(3)
	fmt.Println(data2)

}
