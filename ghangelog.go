package main

import (
	"encoder/json"
	"flag"
	"fmt"
	"net/http"
)

var hostName = flag.String("host", "localhost", "Hostname or IP you want to run this service on")
var portNumber = flag.Int("port", 8080, "Port you want this service to listen on (default 8080)")
var fileToUpdate = flag.String("file", "", "Wiki file to update")
var wikiUrl = flag.String("file", "", "Wiki file to update")

func main() {
	// Listen to the given host:port

	// When a POST is made, read the content and unmarshal it to a struc

	// if state is "closed" check "merge_commit_sha" if it's empty or not

	// Read the "body" and split("\r\n") it into a list of string

	// check if the project.wiki folder exist: If it doesn't exists git clone, if it exists
	// git pull

	// open fileToUpdate, read all the text and split by lines

	// check version number from the given URL: if it's the same of the first line of the wiki,
	// append the body strings where is the first empty line, else create a new version section
	// and append the body strings there.
}
