package main

import (
	"fmt"
	"mapreduce"
	"os"
	"strings"
	"unicode"
	"log"
	"strconv"
)

// The mapping function is called once for each piece of the input.
// In this framework, the key is the name of the file that is being processed,
// and the value is the file's contents. The return value should be a slice of
// key/value pairs, each represented by a mapreduce.KeyValue.
func mapF(document string, value string) (res []mapreduce.KeyValue) {
	// TODO: you have to write this function
	// value: content of the document

	// if string is empty
	// return empty res
	if len(value) == 0 {
		log.Fatal("file ", document," is empty.")
		return res
	}

	// count words in this file
	wordCount := countWords(value)

	//if there is no valid words in this file
	//return empty res
	if wordCount == nil {
		return res
	}

	for w,c := range wordCount {
		res = append(res, mapreduce.KeyValue{ Key: w, Value: strconv.Itoa(c) })
	}
	return res

}

func countWords(value string) (map[string]int) {
	// use a map to save word/count pair
	// return this map
	wcount := make(map[string]int)

	//define the split rule
	//if the character is not letter and not number, it is a partition symbol
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	words := strings.FieldsFunc(value,f)

	//if words is nil, there is no valid words in this file
	if words == nil {
		return wcount
	}
	for _, w := range words {
		if _, ok := wcount[w]; ok {
			wcount[w] += 1
		} else {
			wcount[w] = 1
		}
	}
	return wcount

}

// The reduce function is called once for each key generated by Map, with a
// list of that key's string value (merged across all inputs). The return value
// should be a single output value for that key.
func reduceF(key string, values []string) string {
	// TODO: you also have to write this function
	// key: keyword, values: []count of this key word in all files
	sum := 0
	for _, v := range values {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal(key, " has count ", v, "is not valid")
		}
		sum += i
	}
	return strconv.Itoa(sum)
}

// Can be run in 3 ways:
// 1) Sequential (e.g., go run wc.go master sequential x1.txt .. xN.txt)
// 2) Master (e.g., go run wc.go master localhost:7777 x1.txt .. xN.txt)
// 3) Worker (e.g., go run wc.go worker localhost:7777 localhost:7778 &)
func main() {
	if len(os.Args) < 4 {
		fmt.Printf("%s: see usage comments in file\n", os.Args[0])
	} else if os.Args[1] == "master" {
		var mr *mapreduce.Master
		if os.Args[2] == "sequential" {
			mr = mapreduce.Sequential("wcseq", os.Args[3:], 3, mapF, reduceF)
		} else {
			mr = mapreduce.Distributed("wcseq", os.Args[3:], 3, os.Args[2])
		}
		mr.Wait()
	} else {
		mapreduce.RunWorker(os.Args[2], os.Args[3], mapF, reduceF, 100)
	}
}