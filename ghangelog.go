package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
var output bytes.Buffer

func parseGhPost(rw http.ResponseWriter, request *http.Request) {
	// When a POST is made, read the content and unmarshal it to a struc
	fmt.Println("Decoding new payload ...")
	payload_decoder := json.NewDecoder(request.Body)

	var payload PayLoad
	err_payload := payload_decoder.Decode(&payload)

	if err_payload != nil {
		log.Fatalf("Error decoding payload: %s", err_payload)
	}

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
				fmt.Printf("Executing: %s\n", clone_command)
				cmd := exec.Command("sh", "-c", clone_command)
				cmd.Stdout = &output
				err := cmd.Run()

				if err != nil {
					log.Fatalf("Error cloning git repository (%s): %s", clone_command, err)
				}
			} else {
				// Wiki folder exists, we will have to pull to get updates
				pull_command := fmt.Sprintf("git -C %s pull", configuration.WikiPath)
				fmt.Printf("Executing: %s\n", pull_command)
				cmd := exec.Command("sh", "-c", pull_command)
				cmd.Stdout = &output
				err := cmd.Run()

				if err != nil {
					log.Fatalf("Error pulling git repository (%s): %s", pull_command, err)
				}
			}

			// Check the VersionUrl and get the current released version of the software.
			// This number will be used to determinate if the current version is already in the
			// wiki changelog or if we have to add a new version section.
			fmt.Printf("Getting current version from %s ...\n", configuration.VersionUrl)
			version_resp, err := http.Get(configuration.VersionUrl)

			if err != nil {
				log.Fatalf("Error getting current version: %s", err)
			}

			defer version_resp.Body.Close()
			version_content, err := ioutil.ReadAll(version_resp.Body)
			version := strings.Split(string(version_content), "\n")[0]

			// open fileToUpdate, read all the text and split by lines
			fmt.Printf("Reading from %s ...\n", configuration.WikiChangelogPath)
			content, err := ioutil.ReadFile(configuration.WikiChangelogPath)

			if err != nil {
				log.Fatalf("Error reading from %s: %s", configuration.WikiChangelogPath, err)
			}

			wiki_lines := strings.Split(string(content), "\n")

			// Properly format the changelog rows to append to the wiki page
			var changelog_lines []string

			for _, bl := range body_lines {
				line_elements := strings.Split(bl, " ")
				changelog_line := fmt.Sprintf("[#%s](../issues/%s) %s", line_elements[0][1:],
					line_elements[0][1:], strings.Join(line_elements[1:], " "))
				changelog_lines = append(changelog_lines, changelog_line)
			}

			// Check if the released version is the same we have at the beginning of the wiki
			fmt.Println("Parsing wiki lines ...")
			if wiki_lines[0] == fmt.Sprintf("## %s", version) {
				// Find the first empty line and insert the body_lines there
				for ln, line := range wiki_lines {
					if line == "" {
						wiki_lines = append(wiki_lines[:ln],
							append(changelog_lines, wiki_lines[ln:]...)...)

						fmt.Printf("Writing changes to %s ...\n", configuration.WikiChangelogPath)
						wiki_output := []byte(strings.Join(wiki_lines, "\n"))
						err := ioutil.WriteFile(configuration.WikiChangelogPath, wiki_output,
							os.ModeAppend)

						if err != nil {
							log.Fatalf("Error writing to %s: %s", configuration.WikiChangelogPath,
								err)
						}

						// Commit the edited wiki
						commit_command := fmt.Sprintf("git -C %s commit -a -m '%s'",
							configuration.WikiPath, payload.Request.Title)
						fmt.Printf("Executing: %s\n", commit_command)
						cmd := exec.Command("sh", "-c", commit_command)
						cmd.Stdout = &output
						err = cmd.Run()

						if err != nil {
							log.Fatal("Error in git (%s): %s", commit_command, err)
						}

						// Push the wiki page on GitHub
						push_command := fmt.Sprintf("git -C %s push origin master",
							configuration.WikiPath)
						fmt.Printf("Executing: %s\n", push_command)
						cmd = exec.Command("sh", "-c", push_command)
						cmd.Stdout = &output
						err = cmd.Run()

						if err != nil {
							log.Fatal("Error in git (%s): %s", push_command, err)
						}

						break
					}
				}

			} else {
				// Add a new section with the new version at the beginning of the file
				fmt.Println(body_lines)
			}
		}
	}
}

func main() {
	// Read configuration from config.json
	fmt.Println("Loading configuration from config.json ...")
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&configuration)

	if err != nil {
		log.Fatalf("Error reading from config.json: %s", err)
	}

	// Listen to the given host:port
	fmt.Printf("Listening on %s:%d\n", configuration.Host, configuration.Port)
	http.HandleFunc("/", parseGhPost)
	http.ListenAndServe(fmt.Sprintf("%s:%d", configuration.Host, configuration.Port), nil)
}
