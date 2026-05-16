package main

import (
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"os"
)

type Tag struct {
	EPC string
}

func main() {

	file, err := os.Open("tags.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	rows, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	var tags []Tag

	for _, row := range rows {
		tags = append(tags, Tag{EPC: row[1]})
	}

	out, err := os.Create("tags.gob")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	encoder := gob.NewEncoder(out)
	encoder.Encode(tags)

	fmt.Println("tags.gob created successfully")
}
