package main

import "fmt"

func main() {
	filename := "pgmkt.dat"
	// db2file(filename)
	pgcs, err := readMktData(filename)
	fmt.Println(err)
	fmt.Println(len(pgcs))
	for i := 0; i < 100; i++ {
		fmt.Println(pgcs[i])
	}
}
