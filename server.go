package main
import (
        "encoding/json"
	"fmt"
	"net/http"
	"strings"
        "github.com/cloudfoundry-community/go-cfclient"
)

var policytoGroup map[string]string

type SpaceGroup struct {
	Space    string `json:"space"`
	Endpoint string `json:"endpoint"`
}

func sg(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		//defer func() {
		//if r := recover(); r != nil {
		//w.Write([]byte("49920-50175"))
		//}
		//}()
		//todo: no talking with pg apis, find name from cf and parse it
		space := r.URL.Query().Get("space")

		policy := GetPolicy(space)
		res := policytoGroup[policy]
		if res == "" {
			res = "49920-50175"
		}
		w.Write([]byte(res))
		//http.Error(w, "cannot find matching space", http.StatusBadRequest)
		return

	case "POST":
		var req SpaceGroup
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Endpoint == "" || req.Space == "" {
			http.Error(w, "post data error", http.StatusBadRequest)
			return
		}
		fmt.Println("got post data %s, %s", req.Space, req.Endpoint)
	}

}

func config() {
	//conatins the hard coded stuff
	policytoGroup["blue"] = "49920-50175"
	policytoGroup["green"] = "50176-50431"
	policytoGroup["red"] = "50432-50687"
        policytoGroup["empty"] = ""
}

func GetPolicy(spaceid string) string {
	c := &cfclient.Config{
		ApiAddress:        "https://api.cf.plumgrid.com",
		Username:          "admin",
		Password:          "plumgrid",
		SkipSslValidation: true,
	}
        fmt.Println(spaceid)
	client, _ := cfclient.NewClient(c)
	spaces, _ := client.ListSpaces()
	for _, s := range spaces {
               if (s.Guid == spaceid) {
                        fmt.Println("bingo")
                        return parse(s.Name)
               }
	}
	return "empty"
}

func parse(name string) string {
	//hard coded
	switch {
	case strings.Contains(name, "dev"):
		return "blue"
	case strings.Contains(name, "int"):
		return "green"
	case strings.Contains(name, "prod"):
		return "red"
	}
	return "blue"

}

func main() {

	policytoGroup = make(map[string]string)
	config()
	mux := http.NewServeMux()
	mux.HandleFunc("/spacegroup", sg)
	http.ListenAndServe(":8000", mux)

}
