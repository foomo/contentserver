package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foomo/contentserver/server/repo/content"
	"io/ioutil"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, obj interface{}) {
	fmt.Fprint(w, toJson(obj))
}

func ToJSON(obj interface{}) string {
	return toJson(obj)
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

func GetRepo(URL string, obj map[string]*content.RepoNode) (ok bool, err error) {
	// add proper error handling
	response, err := http.Get(URL)
	if err != nil {
		return false, err
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return false, errors.New(fmt.Sprintf("Bad HTTP Response: %v", response.Status))
		} else {
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return false, err
			} else {
				fmt.Printf("json string %s", string(contents))
				jsonErr := json.Unmarshal(contents, &obj)
				if jsonErr != nil {
					panic(jsonErr)
					return false, jsonErr
				} else {
					return true, nil
				}
			}
		}
	}
}

func Get(URL string, obj interface{}) (ok bool, err error) {
	// add proper error handling
	response, err := http.Get(URL)
	if err != nil {
		return false, err
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return false, errors.New(fmt.Sprintf("Bad HTTP Response: %v", response.Status))
		} else {
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return false, err
			} else {
				fmt.Printf("json string %s", string(contents))
				jsonErr := json.Unmarshal(contents, &obj)
				if jsonErr != nil {
					panic(jsonErr)
					return false, jsonErr
				} else {
					return true, nil
				}
			}
		}
	}
}
