/*
NUSAIBA RAHMAN
UC BERKELEY CS61C 
All rights and property belong to UC Berkeley. 

A file server uses some protocol to connect to other computers on a network. 
When it gets a request from some other computer on the network, it processes 
the request (typically as a path to a file in the form of a URL), and returns 
back the data of the file which is local on the server. The protocol allows 
for a well defined communication method to exist between the different 
computers on the network. The common protocol which used to be used everywhere 
for this is http. 

The problem for many of these sites is that disk reads are really slow so 
when many users are trying to access a file, it can take a long time for the 
request to get a response. To combat this, many file servers implement caching 
so that subsequent reads to the same file can be faster.

One problem a server may face is a lot of disk reads at once. This can make
the response to a request take a long time. Sometimes the disk may just never 
respond though is highly unlikely. Because of this, servers implement a 
timeout system to ensure that requests are answered.

Go is the language of choice becuase it was designed to support concurrency.

This is the implementation of a file server's file cacher. 

*/

package main

import (
	"flag"
	"fmt"
	"github.com/61c-teach/sp19-proj5-userlib"
	"net/http"
	"log"
	"strings"
	"time"
)

// This is the handler function which will handle every request other than cache specific requests.
func handler(w http.ResponseWriter, r *http.Request) {
	// Note that we will be using userlib.ReadFile we provided to read files on the system.
	// The path to the file is given by r.URL.Path and will be the path to the string.

	filename := r.URL.Path[0:]
	res := getFile(filename)
	filename = res.filename
	response := res.responseData

	if res.responseError == nil { /*was able to access the chache*/

		// This will automatically set the right content type for the reply as well.
		w.Header().Set(userlib.ContextType, userlib.GetContentType(filename))
		// Set the correct header code for a success since we should only succeed at this point.
		w.WriteHeader(userlib.SUCCESSCODE) // Make sure you write the correct header code so that the tests do not fail!
		// Write the data which is given to us by the response.
		w.Write(response)

	} else { //was not able to access the cache

		if res.responseError.Error() == userlib.TimeoutString {
			http.Error(w, userlib.TimeoutString, userlib.TIMEOUTERRORCODE)
			return
		} else {
			fmt.Printf("GOT A FILEREAD\n")
			http.Error(w, userlib.FILEERRORMSG, userlib.FILEERRORCODE)
			return
		}

	}
}

// This function will handle the requests to acquire the cache status.
func cacheHandler(w http.ResponseWriter, r *http.Request) {
	// Sets the header of the request to a plain text format since we are just dumping information about the cache.
	// Set temporary "fake" filename which will get the correct content type.
	w.Header().Set(userlib.ContextType, userlib.GetContentType("cacheStatus.txt"))
	// Set the success code to the proper success code since the action should not fail.
	w.WriteHeader(userlib.SUCCESSCODE)
	// Get the cache status string from the getCacheStatus function.
	w.Write([]byte(getCacheStatus()))
}

// This function will handle the requests to clear/restart the cache.
func cacheClearHandler(w http.ResponseWriter, r *http.Request) {
	// Sets the header of the request to a plain text format since we are just dumping information about the cache.
	// Note that we are just putting a "fake" filename which will get the correct content type.
	w.Header().Set(userlib.ContextType, userlib.GetContentType("cacheClear.txt"))
	// Set the success code to the proper success code since the action should not fail.
	w.WriteHeader(userlib.SUCCESSCODE)
	// Get the cache status string from the getCacheStatus function.
	w.Write([]byte(CacheClear()))
}

// The structure used for responding to file requests.
// It contains the file contents (if there is any)
// or the error returned when accessing the file.
type fileResponse struct {
	filename string
	responseData []byte
	responseError error
	responseChan chan *fileResponse
}

// To request files from the cache, we send a message that
// requests the file and provides a channel for the return
// information.
type fileRequest struct {
	filename string
	response chan *fileResponse
}

// Port of the server to run on
var port int
// Capacity of the cache in Bytes
var capacity int
// Timeout for file reads in Seconds.
var timeout int
// Working directory of the server
var workingDir string

// The channel to pass file read requests to. This is how you will get a file from the cache.
var fileChan = make(chan *fileRequest)
// The channel to pass a request to get back the capacity info of the cache.
var cacheCapacityChan = make(chan chan string)
// The channel where a bool passed into it will cause the OperateCache function to be closed and all of the data to be cleared.
var cacheCloseChan = make(chan bool)

// A wrapper function that does the actual getting of the file from the cache.
func getFile(filename string) (response *fileResponse) {
	// Sanity check: The requested file
	// should be made relative (strip out leading "/" characters,
	// then have a "./" put on the start, and if there is ever the
	// string "/../", replace it with "/", the string "\/" should
	// be replaced with "/", and finally any instances of "//" (or
	// more) should be replaced by a single "/".
	// Requests with is just "/", return the file "./index.html"
	// Return a file not found error if after `timeout`
	// seconds if there is no response from the cache.

	og := filename
	repeater := -1
	for {
		filename = strings.Replace(filename, "\\/", "/", repeater)
		filename = strings.Replace(filename, "/../", "/", repeater)
		filename = strings.Replace(filename, "//", "/", repeater)

		if filename == og {
			break
		}
		og = filename
	}

	if len(filename) != 0 && filename[0] == '/' {
		filename = "." + filename
	}
	l := len(filename) - 1

	if len(filename) != 0 && filename[l] == '/' {
		filename += "index.html"
	}

	// Make a request on the fileChan and wait for a response to be issued from the cache.
	// Makes the file request object.
	request := fileRequest{filename, make(chan *fileResponse)}
	// Sends a pointer to the file request object to the fileChan so the cache can process the file request.
	fileChan <- &request
	// Returns the result (from the fileResponse channel)
	return <- request.response //something it's dequed from the channel - returns that
}

// This function returns a string of the cache current status.
// It will just make a request to the cache asking for the status.
func getCacheStatus() (response string) {
	// Make a channel for the response of the Capacity request.
	responseChan := make(chan string)
	// Send the response channel to the capacity request channel.
	cacheCapacityChan <- responseChan
	// Return the reply.
	return <- responseChan
}

// This function will tell the cache that it needs to close itself.
func CacheClear() (response string) {
	// Send the response channel to the capacity request channel.
	cacheCloseChan <- true
	// We should only return to here once we are sure the currently open cache will not process any more requests.
	// This is because the close channel is blocking until it pulls the item out of there.
	// Now that the cache should be closed, relaunch the cache.
	go operateCache()
	return userlib.CacheCloseMessage
}

type cacheEntry struct {
	filename string
	data []byte

	erro error

}

/*CORE CACHE LOGIC*/
//Maps with all chache entries and extra channels for concurrency. 

func operateCache() {
	// Make a file map for the cache entries.
	myCacheMap := make(map[string]cacheEntry)
	currCach := 0
	addingtoCacheChan := make(chan cacheEntry)

	// Made filemap, now handle requests
	for {
		// Select what we want to do based on what is in different cache channels.
		select {
		case fileReq := <-fileChan:
			// Handle a file request here.
			if val, contains := myCacheMap[fileReq.filename]; contains {
				fileReq.response <- &fileResponse{fileReq.filename,
					val.data,
					nil,
					nil}

			} else {
				//if the disk take some about of time and causes a timeout, still put the file into the cache, but don't respond
				//put it in the cache
				go func(re fileRequest) { //reading from the disk nonblocking way
					//LAUNCHING EITHER FILE ROUTINE OR TIMEOUT
					gettingFileReadsFromDisk := make(chan fileResponse)
					go func() {
						d, e := userlib.ReadFile(workingDir, re.filename)
						gettingFileReadsFromDisk <- fileResponse{re.filename, d, e, nil}
					}()

					timed := 0

					select {
					case res := <-gettingFileReadsFromDisk:
						//if res.responseError != nil {
						addingtoCacheChan <- cacheEntry{res.filename, res.responseData, res.responseError}

						fileReq.response <- &fileResponse{res.filename,
							res.responseData,
							res.responseError,
							nil}
						//}
					case <-time.After(time.Duration(timeout) * time.Second):
						//create reponse object that says i times out
						fileReq.response <- &fileResponse{fileReq.filename,
							nil,
							fmt.Errorf(userlib.TimeoutString),
							nil}
						timed = 1

					}
					if timed == 1 {
						res := <-gettingFileReadsFromDisk
						addingtoCacheChan <- cacheEntry{res.filename, res.responseData, res.responseError}

						fileReq.response <- &fileResponse{res.filename,
							res.responseData,
							res.responseError,
							nil}
					}
				}(*fileReq)
			}

		case addedtoCache := <-addingtoCacheChan:

			if _, contains := myCacheMap[addedtoCache.filename]; contains {

			} else {

				if addedtoCache.erro != nil {

				} else {
					if len(addedtoCache.data) <= capacity && addedtoCache.erro == nil {

						if capacity-currCach >= len(addedtoCache.data) {
							if _, contains := myCacheMap[addedtoCache.filename]; contains {

							} else {
								myCacheMap[addedtoCache.filename] = addedtoCache
								currCach += len(addedtoCache.data)
							}
						} else {

								if capacity >= len(addedtoCache.data) {
								for k, v := range myCacheMap {
									if capacity-(currCach-len(v.data)) >= len(addedtoCache.data) {
										delete(myCacheMap, k)
										currCach -= len(v.data)
										myCacheMap[addedtoCache.filename] = addedtoCache
										currCach += len(addedtoCache.data)
										break
									} else {
										delete(myCacheMap, k)
										currCach -= len(v.data)
									}
								}

							} else {

							}
						}
					}
				}
			}

		case cacheReq := <- cacheCapacityChan:
			cacheReq <- fmt.Sprintf(userlib.CapacityString, len(myCacheMap), currCach, capacity)

		case <- cacheCloseChan:
			// Exit the cache.
			for k := range myCacheMap {
				delete(myCacheMap, k)
			}
			currCach = 0
			return
		}

	}
}


// This functions when you do `go run server.go`. It will read and parse the command line arguments, set the values
// of some global variables, print out the server settings, tell the `http` library which functions to call when there
// is a request made to certain paths, launch the cache, and finally listen for connections and serve the requests
// which a connection may make. When it services a request, it will call one of the handler functions depending on if
// the prefix of the path matches the pattern which was set by the HandleFunc.
func main(){
	// Initialize the arguments when the main function is ran. This is to setup the settings needed by
	// other parts of the file server.
	flag.IntVar(&port, "p", 8080, "Port to listen for HTTP requests (default port 8080).")
	flag.IntVar(&capacity, "c", 100000, "Number of bytes to allow in the cache.")
	flag.IntVar(&timeout, "t", 2, "Default timeout (in seconds) to wait before returning an error.")
	flag.StringVar(&workingDir, "d", "public_html/", "The directory which the files are hosted in.")
	// Parse the args.
	flag.Parse()
	// Say that we are starting the server.
	fmt.Printf("Server starting, port: %v, cache size: %v, timout: %v, working dir: '%s'\n", port, capacity, timeout, workingDir)
	serverString := fmt.Sprintf(":%v", port)

	// Set up the service handles for certain pattern requests in the url.
	http.HandleFunc("/", handler)
	http.HandleFunc("/cache/", cacheHandler)
	http.HandleFunc("/cache/clear/", cacheClearHandler)

	// Start up the cache logic...
	go operateCache()

	// This starts the web server and will cause it to continue to listen and respond to web requests.
	log.Fatal(http.ListenAndServe(serverString, nil))
}