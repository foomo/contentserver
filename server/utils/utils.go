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

func Get(URL string) (data []byte, err error) {
	response, err := http.Get(URL)
	if err != nil {
		return data, err
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return data, errors.New(fmt.Sprintf("Bad HTTP Response: %v", response.Status))
		} else {
			return ioutil.ReadAll(response.Body)
		}
	}
}
