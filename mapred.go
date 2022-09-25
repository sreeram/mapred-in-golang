package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type mapJob struct {
	wordsBag   *[]string
	startIndex int
	length     int
}

type reduceJob struct {
	counts map[string]int
}

func mapper(c1 chan mapJob, c2 chan reduceJob, wg *sync.WaitGroup) {
	//for job := range c1 {
	for {
		job := <-c1
		//fmt.Println("mapper: ", job.startIndex)
		go mapfunc(job, c2)
		wg.Add(1)
	}
}

func mapfunc(job mapJob, c chan reduceJob) {
	j := *job.wordsBag
	start := job.startIndex
	ct := job.length

	counts := make(map[string]int)
	for _, w := range j[start : start+ct+1] {
		counts[w]++
	}
	//fmt.Println("mapfunc(", start, ":", start+ct, ")")
	c <- reduceJob{counts}
	//fmt.Println(counts)
}

func myMin(a int, b int) int {
	min := b
	if a < b {
		min = a
	}
	return min
}
func reducer(c2 chan reduceJob, finalCounts map[string]int, wg *sync.WaitGroup) {
	for {
		input := <-c2
		counts := input.counts
		for w, ct := range counts {
			finalCounts[w] += ct
		}
		wg.Done()
	}
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func clearString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, " ")
}

func main() {

	if len(os.Args) != 2 {
		log.Fatal("Need to supply input file as argument")
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	words := 0
	var buffer []string

	var wg sync.WaitGroup
	for scanner.Scan() {
		input := scanner.Text()
		split := strings.Fields(clearString(input))
		for _, s := range split {
			buffer = append(buffer, strings.ToLower(s))
		}
		words += len(split)
	}
	fmt.Println("#Words: ", len(buffer))

	now := time.Now()
	startTimeNanos := now.UnixNano()
	numCPU := runtime.NumCPU()
	//fmt.Println("NumCPU: ", numCPU)
	c1 := make(chan mapJob, numCPU)
	c2 := make(chan reduceJob, numCPU)

	go mapper(c1, c2, &wg)

	numWordsPerMap := 200000
	for i := 0; i < len(buffer); i += numWordsPerMap {
		c1 <- mapJob{&buffer, i, myMin(numWordsPerMap, len(buffer)-i)}
	}

	finalCounts := make(map[string]int)
	go reducer(c2, finalCounts, &wg)

	wg.Wait()
	now = time.Now()
	endTimeNanos := now.UnixNano()
	fmt.Println("mapred(", "numCPU=", numCPU, " wordsPerMap=", numWordsPerMap, " took ", (endTimeNanos-startTimeNanos)/1000, "micros")
	//fmt.Println(finalCounts)

	now = time.Now()
	startTimeNanos = now.UnixNano()
	counts := make(map[string]int)
	for _, w := range buffer {
		counts[w]++
	}
	now = time.Now()
	endTimeNanos = now.UnixNano()
	//fmt.Println(counts)
	fmt.Println("serial(", "numCPU=", numCPU, " wordsPerMap=", numWordsPerMap, " took ", (endTimeNanos-startTimeNanos)/1000, "micros")
}
