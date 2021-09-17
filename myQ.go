package pmoo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"net/http"

	"github.com/donomii/pmoo"
)

func Receiver(url string, callback func([]byte)) {
	for {
		resp, err := http.Get(url + "/subscribe/main")
		if err != nil {
			//log.Fatalln(err)
		} else {
			data, _ := ioutil.ReadAll(resp.Body)
			callback(data)
		}
	}
}

func Message(url string, mess interface{}) {
	json, _ := json.Marshal(mess)
	Send(url, json)
}

func Send(url string, data []byte) {
	http.Post(url+"/publish/main", "who/cares", bytes.NewReader(data))
}

func StoreObject(url, id string, m pmoo.Object) {
	b, _ := json.Marshal(m)
	http.Post(url+"/store/"+id, "who/cares", bytes.NewReader(b))
}

func FetchObject(url, id string) *pmoo.Object {
	resp, err := http.Get(url + "/subscribe/main")
	if err != nil {
		//log.Fatalln(err)
	}
	data, _ := ioutil.ReadAll(resp.Body)

	retrievedVal := new(pmoo.Object)
	json.Unmarshal(data, retrievedVal)
	return retrievedVal
}

/*
func DeleteObject(id string) {
	err := KVstore.Delete(id)
	if err != nil {
		panic(err)
	}
}
*/
