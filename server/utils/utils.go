package utils

import (
	//"encoding/json"
	"fmt"
	"github.com/foomo/ContentServer/server/jjson"
	"io/ioutil"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, obj interface{}) {
	fmt.Fprint(w, toJson(obj))
}

func toJson(obj interface{}) string {
	//b, err := json.MarshalIndent(obj, "", "\t")
	b, err := jjson.Marshal(obj)
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
	jjson.Unmarshal(extractJsonFromRequest(r), obj)
}

func Get(URL string, obj interface{}) {
	// add proper error handling
	response, err := http.Get(URL)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			fmt.Errorf("Bad HTTP Response: %v", response.Status)
		}
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}
		// fmt.Printf("json string %s", string(contents))
		jsonErr := jjson.Unmarshal(contents, &obj)
		if jsonErr != nil {
			fmt.Println("wtf", jsonErr)
		}
	}

}
