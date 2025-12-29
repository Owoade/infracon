package utils

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Map[T comparable](arr []T, cb func(T) T) []T {
	newArr := []T{}
	for _, each := range arr {
		mappedValue := cb(each)
		newArr = append(newArr, mappedValue)
	}
	return newArr
}