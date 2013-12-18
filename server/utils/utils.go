package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, obj interface{}) {
	fmt.Fprint(w, toJson(obj))
}

func toJson(obj interface{}) string {
	//b, err := json.MarshalIndent(obj, "", "\t")
	b, err := json.Marshal(obj)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func extractJsonFromRequestFileUpload(r *http.Request) []byte {
	file, _, err := r.FormFile("request")
	if err != nil {
		fmt.Println(err, r)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	return data
}
func extractJsonFromRequest(r *http.Request) []byte {
	bytes := []byte(r.PostFormValue("request"))
	return bytes
}

func PopulateRequest(r *http.Request, obj interface{}) {
	json.Unmarshal(extractJsonFromRequest(r), obj)
}

func Get(URL string, obj interface{}) (ok bool, err error) {
	// add proper error handling
	response, err := http.Get(URL)
	defer response.Body.Close()
	if err != nil {
		return false, err
	} else {
		if response.StatusCode != http.StatusOK {
			return false, errors.New(fmt.Sprintf("Bad HTTP Response: %v", response.Status))
		} else {
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return false, err
			} else {
				// fmt.Printf("json string %s", string(contents))
				jsonErr := json.Unmarshal(contents, &obj)
				if jsonErr != nil {
					return false, jsonErr
				} else {
					return true, nil
				}
			}
		}
	}
}
