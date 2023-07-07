package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

var ret []map[string]interface{}

const url = "https://jsonplaceholder.typicode.com/users"

func v1() {
	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	// Check the response status code
	if resp.StatusCode != 200 {
		panic(fmt.Sprintf("status code error: %d %s", resp.StatusCode, resp.Status))
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// convert the response body to a string
	err = json.Unmarshal(body, &ret)
	if err != nil {
		panic(err)
	}

	// Print the response body indented
	prettyJSON, err := json.MarshalIndent(ret, "", "    ")
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%s\n", string(prettyJSON))
	_ = prettyJSON
}

func v2() {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// convert the response body to a string
	err = json.Unmarshal(body, &ret)
	if err != nil {
		panic(err)
	}

	// Print the response body indented
	prettyJSON, err := json.MarshalIndent(ret, "", "    ")
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%s\n", string(prettyJSON))
	_ = prettyJSON

}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
// (from https://golangcode.com/download-a-file-with-progress/)
func downloadFile(filepath string) (err error) {

	// Create the ouput file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	v2()

	fmt.Println("Done")
}
