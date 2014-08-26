package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Configuration struct {
	Token             string `json:"token"`
	Host              string `json:"host"`
	Port              int32  `json:"port"`
	WikiGitUrl        string `json:"wiki_git_url"`
	GitUsername       string `json:"git_username"`
	WikiChangelogPath string `json:"wiki_changelog_path"`
	WikiPath          string `json:"wiki_path"`
	VersionUrl        string `json:"version_url"`
}

type PullRequest struct {
	State          string `json:"state"`
	Body           string `json:"body"`
	MergeCommitSha string `json:"merge_commit_sha"`
	Title          string `json:"title"`
}

type PayLoad struct {
	Request PullRequest `json:"pull_request"`
}

var configuration = Configuration{}

func parseGhPost(rw http.ResponseWriter, request *http.Request) {
	// When a POST is made, read the content and unmarshal it to a struc
	payload_decoder := json.NewDecoder(request.Body)

	var payload PayLoad
	err_payload := payload_decoder.Decode(&payload)

	if err_payload != nil {
		panic(err_payload)
	}

	fmt.Println(payload)

	// if state is "closed" check "merge_commit_sha" if it's empty or not
	if payload.Request.State == "closed" {
		if payload.Request.MergeCommitSha != "" {
			// Read the "body" and split("\r\n") it into a list of string
			body_lines := strings.Split(payload.Request.Body, "\r\n")

			// check if the project.wiki folder exist: If it doesn't exists git clone, if it exists
			// git pull
			_, err := os.Stat(configuration.WikiPath)

			if err != nil {
				// Wiki folder doesn't exists, we will have to clone from git
				clone_command := fmt.Sprintf("git clone https://%s:%s@%s",
					configuration.GitUsername, configuration.Token, configuration.WikiGitUrl)
				exec.Command("sh", "-c", clone_command)
			} else {
				// Wiki folder exists, we will have to pull to get updates
				pull_command := fmt.Sprintf("git -C %s pull", configuration.WikiPath)
				exec.Command("sh", "-c", pull_command)
			}

			// Check the VersionUrl and get the current released version of the software.
			// This number will be used to determinate if the current version is already in the
			// wiki changelog or if we have to add a new version section.
			version_resp, err := http.Get(configuration.VersionUrl)

			if err != nil {
				panic(err)
			}

			defer version_resp.Body.Close()
			version_content, err := ioutil.ReadAll(version_resp.Body)
			version := strings.Split(string(version_content), "\n")[0]

			// open fileToUpdate, read all the text and split by lines
			content, err := ioutil.ReadFile(configuration.WikiChangelogPath)

			if err != nil {
				panic(err)
			}

			wiki_lines := strings.Split(string(content), "\n")

			// Check if the released version is the same we have at the beginning of the wiki
			if wiki_lines[0] == fmt.Sprintf("## %s", version) {
				// Find the first empty line and insert the body_lines there
				for ln, line := range wiki_lines {
					if line == "" {
						wiki_lines = append(wiki_lines[:ln],
							append(body_lines, wiki_lines[ln:]...)...)
						break
					}
				}

			} else {
				// Add a new section with the new version at the beginning of the file
				fmt.Println(body_lines)
			}

			fmt.Println(wiki_lines)
		}
	}
}

func main() {
	// Read configuration from config.json
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&configuration)

	if err != nil {
		fmt.Println("error:", err)
	}

	// Listen to the given host:port
	http.HandleFunc("/", parseGhPost)
	http.ListenAndServe(fmt.Sprintf("%s:%d", configuration.Host, configuration.Port), nil)
}
