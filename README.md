Getting Going with Go

The best starting point to learn Go is a quick read of https://www.golang-book.com/books/intro. Go is a lot like C. It has got pointers, structs, etc.
Concurrency constructs in Go

Obviously, the intention of the project is to learn the concurrency aspects of the Go language. The two key concurrency constructs in Go are Goroutines and Channels.

You can read about both at https://www.golang-book.com/books/intro/10

A bit more nuanced write-up is https://go.dev/doc/effective_go#concurrency
Goroutine

Goroutine is like a lightweight thread that you can kick off by prefixing a function invocation with the keyword go (e.g: go foo()). The next line in the program continues on as if it is unaware of the thread that was just created. The new thread finishes whenever it does.  

package main
import "fmt"

func f(n int) {
  for i := 0; i < 10; i++ {
    fmt.Println(n, ":", i)
  }
}

func main() {
  go f(0)            // A goroutine was called. f() starts executing 
  var input string   // life goes on in parallel
  fmt.Scanln(&input) // pause program for input from terminal, f() can end
}
This program consists of two goroutines. The first goroutine is implicit and is the main function itself. The second goroutine is created when we call go f(0). Normally when we invoke a function our program will execute all the statements in a function and then return to the next line following the invocation. With a goroutine we return immediately to the next line and don't wait for the invoked function to complete. 
 
This is why the call to the Scanln function has been included; without it the program would exit before being given the opportunity to print all the numbers. Goroutines are lightweight and we can easily create thousands of them. 

The Golang runtime kicks off a little heap and some stack to get that “thread” going pretty much instantaneously. This is the most significant simplification of thread programming ever. Pretty awesome. 

Ok, so you got that? Great!

Channel

So you create threads super easily, right. Now you need to make them co-operate and do things with some orchestration. Golang Channel is a mechanism for sequential programs and goroutines to send typed messages to each other. It is a Queue mechanism! You can send or wait for messages on a Channel.


Declaring a Channel is easy!

<variableName> chan <Type>

Example:

C chan string  // you can exchange string messages on a channel name C
myQueue chan [string]map[]  // Pass a Map of String -> int values

You push a message with C <- “foo” or read a message from the channel as s := <- C


Channels provide a way for two goroutines to communicate with one another and synchronize their execution. Here is an example program using channels:
package main

import (
  "fmt"
  "time"
)

func pinger(c chan string) {
  for i := 0; ; i++ {
    c <- "ping"
  }
}

func printer(c chan string) {
  for {
    msg := <- c
    fmt.Println(msg)
    time.Sleep(time.Second * 1)
  }
}

func main() {
  var c chan string = make(chan string)

  go pinger(c)
  go printer(c)

  var input string
  fmt.Scanln(&input)
}
This program will print “ping” forever (hit enter to stop it). 
That’s pretty much it, for the basics! 

Definitely read all of https://www.golang-book.com/books/intro/10 and try the examples in a Go IDE. I found the GoLand trial version to be absolutely amazing. 

TODO: Find the other link to GoLang Concurrency tutorial 


What is Map Reduce?

map() concept came first from Lisp language, and LISt Processing is obviously all about Lists. map() function in Lisp takes another function and applies it to every element in the list. 

myList = (1,2,3,4)
As an example, map(myList, square(x) { return x*x;}) would return
myList = (1,4,9,16)

Simple, right?
 
Now the List need not be as simple as integers as in this example. It could be a list of portfolio of stocks. The supplied function may run a risk analysis on each portfolio object in the list. You get the idea. 

So what is Reduce? It is another function that may be optionally invoked to “summarize” across the list. Maybe you want to find the portfolio with the lowest risk? Reduce could sort and return the best portfolio with the least risk.

Map-Reduce got super popular because Google uses it to deliver search results nearly instantaneously. How?

The map operation is heavily data-parallel. It can be applied concurrently on all of the objects in the list, and it would run as fast as the multiple cores on your system can cope with. 
 
Google asks servers on the farm to each lookup search result it has access to and assigns a relevancy score. The Reduce collects the results returned by hundreds of servers and combines them to present the most relevant results (and Ads) 

So you get the Map-Reduce concept. Let’s move on. 


Counting Words in a large Text file

Basically, identify each unique word in a large text file and count the number of occurrences of each word. 

If you can imagine dividing the large text into chunks of say 10K words each and parallelizing this task, you have the right idea. And then the map() function does the tallying of words in its chunk and returns a table of words and frequency in its chunk. 

The reducer simply collects all these partial results and adds them all up to produce the final count. 

That is pretty much what we will do in the Go program. 

Now you want the chunk size to be large, so you get a sufficient bang from each thread/goroutine. You also want to ensure you do not create more threads than the number of cores on your system. Too many threads are sometimes as bad as too few. 

So how do we control that degree of concurrency? We will show how. 

Reading File and creating Chunks

First, let’s write a function that can get rid of punctuation. 

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func clearString(str string) string {
  return nonAlphanumericRegex.ReplaceAllString(str, " ")
}

Now let’s read the input:

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

  for scanner.Scan() {
     input := scanner.Text()
     split := strings.Fields(clearString(input))
     for _, s := range split {
        buffer = append(buffer, strings.ToLower(s))
     }
     words += len(split)
  }
  fmt.Println("#Words: ", len(buffer))
}
Good, now we have a large String Slice in Go with all the words. 

Now we will divide into chunks of words and put Go to work:


func main() {

  // File input and clean up above

  // This counts the numbers of map workers and ensures all of them complete
  // before Go runtime says that the program can exit. Otherwise, the Go
  // may say Bye even before the mappers and reducers have done their jobs
  // This is passed by reference to the mapper and reducer Job schedulers
  // See wg.Wait() 
  var wg sync.WaitGroup

  // Figure out how much parallelism is useful
  numCPU := runtime.NumCPU()
  fmt.Println("NumCPU: ", numCPU)

  // Create two bounded or buffered channels
  // You cannot stuff more than as many messages in each channel
  // since each message activates a Goroutine this limits concurrency!
  c1 := make(chan mapJob, numCPU)
  c2 := make(chan reduceJob, numCPU)

  // Kick of Map function scheduler as Goroutine. But it has not work yet. 
  // It will block on reading c1
  // Once it gets a message on c1, it does the counting work and 
  // posts its partial table of word counts to c2, for the reducer 
  go mapper(c1, c2, &wg)

  // We are chunking to 10K words for each map task
  numWordsPerMap := 10000
  for i := 0; i < len(buffer); i += numWordsPerMap {
     c1 <- mapJob{buffer, i, myMin(numWordsPerMap, len(buffer)-i)}
  }

  // Create a table of word → Integer to hold the final results table 
  finalCounts := make(map[string]int)

  // we initiate the reducer even before we know the mappers are done
  // But the reducer will wait on reading from c2, so we are okay
  go reducer(c2, finalCounts, &wg)


  wg.Wait()
  fmt.Println(finalCounts)
}

So far so good? Now we will look at the mapper and reducer jobs. 

type mapJob struct {   	
  wordsBag   []string   // pass the big bag of all words, by reference.
  startIndex int		// which is the starting index for this chunk
  length     int		// how many words?
}

type reduceJob struct {
  counts map[string]int   // data structure to hold the count tables
}

The mapper scheduler reads chunks from a channel, tallies up words in its chunk and puts its partial table of results in the channel that is going to be read by the reducer.

func mapper(c1 chan mapJob, c2 chan reduceJob, wg *sync.WaitGroup) {
  for job := range c1 {    // block and read chunk/job from channel 
     go mapfunc(job, c2)   // Kick off Goroutine
     wg.Add(1)             // Keep track of each Goroutine
  }
}

func mapfunc(job mapJob, c chan reduceJob) {
  j := job.wordsBag
  start := job.startIndex
  ct := job.length

  counts := make(map[string]int)
  for _, w := range j[start : start+ct+1] {  // build the table for this chunk
     counts[w]++
  }
  //fmt.Println("mapfunc(", start, ":", start+ct, ")")
  c <- reduceJob{counts}				// Push my table to reducer
  //fmt.Println(counts)
}

Ok, for the final piece, the Reducer
func reducer(c2 chan reduceJob, finalCounts map[string]int, wg 
*sync.WaitGroup) {
  for {
     input := <-c2  			// read count table for each chink
     counts := input.counts		
     for w, ct := range counts {	// summarize into the finalCounts table
        finalCounts[w] += ct		// which was passed in as a parameter 
     }						// from main program
     wg.Done()				// we can say that one chunk was processed
  }						// when this loop ends all chunks were
}						// “reduced”
Miscellaneous

Oddly Golang does not have a math.Min that can accept two integers. So I wrote a small utility

func myMin(a int, b int) int {
  min := b
  if a < b {
     min = a
  }
  return min
}


And, put this chunk of code at the beginning so you have all the imports

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
)

Observations

I ran this program on https://norvig.com/big.txt

Times on my M1 Macbook Pro, with chunk size set to 10000

NOTE: The reported elapsed time is in Microseconds, not milliseconds. The text input is small actually, and the work finished in 30 milliseconds! I probably need a much bigger text corpus before it shows any substantive differences in time. 

#Words:  1115031
NumCPU:  10
done. Took  22215 ms

If I throttled the mappers to use half as many cores:

#Words:  1115031
NumCPU:  5
done. Took  24026 ms

#Words:  1115031
NumCPU:  1
done. Took  23441 ms

Hmm.. it looks like my program is quite serial, really????

Now I changed the chunk size to 1000 (10x smaller than above). There should be more mappers and more reduced operation messages

#Words:  1115031
NumCPU:  10
done. Took  40487 ms

So the code is more sensitive now. 

Let’s try throttling the concurrency now
#Words:  1115031
NumCPU:  5
done. Took  40954 ms

#Words:  1115031
NumCPU:  1
done. Took  40457 ms


I just can’t seem to get this to run any faster. 

BUT is it faster than the serial code? Did all these Goroutines and Channels really help?

#Words:  1115031
NumCPU:  10
done. Took  40087 ms
The serial version Took  31738 ms

If I changed the chunk size back to to 10K, it will make the parallel version faster:
#Words:  1115031
NumCPU:  10
done. Took  22773 ms
Serial version Took  29886 ms
Experiment by making chunk = 20K words

#Words:  1115031
NumCPU:  10
done. Took  18010 ms
Serial version Took  29739 ms

50K words

#Words:  1115031
NumCPU:  10
done. Took  12971 ms
Serial version Took  32295 ms

100K words

#Words:  1115031
NumCPU:  10
done. Took  10426 ms
Serial version Took  31009 ms


Conclusion: The MapReduce version is significantly faster when the chunk size is increased to 100K. Multiple Cores helped me when I had moderately sized work for each Map job. When I had too many such jobs, the inefficiencies crept in. 

Epilogue

I had worked a bit with Robert Griesemer, one of the three creators of the Go Programming Language when I worked on the HotSpot Virtual Machine porting to Solaris/Sparc at Sun Microsystems during 1997-99. Robert G. was a key developer in the original Animorphic Systems team and wrote the Hotspot interpreter and the blazing fast “C1” JIT Compiler. Absolutely the clearest-headed VM developer there was. His work was compact, elegant, and always plain fast! He walked away from Java runtimes and eventually to Google. It is a shame that I have not tried out his Go language sooner. The more I use Go, the more I remember those fun days at Mary Avenue, Sunnyvale. 
