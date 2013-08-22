package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, obj interface{}) {
	fmt.Fprint(w, toJson(obj))
}

func toJson(obj interface{}) string {
	b, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func extractJsonFromRequest(r *http.Request) []byte {
	file, _, err := r.FormFile("request")
	if err != nil {
		fmt.Println(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	return data
}

func PopulateRequest(r *http.Request, obj interface{}) {
	json.Unmarshal(extractJsonFromRequest(r), obj)
}

func Get(URL string, obj interface{}) {
	// add proper error handling
	response, err := http.Get(URL)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}
		// fmt.Printf("json string %s", string(contents))
		jsonErr := json.Unmarshal(contents, &obj)
		if jsonErr != nil {
			fmt.Println("wtf", jsonErr)
		}
	}

}
