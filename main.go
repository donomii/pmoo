package pmoo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Property struct {
	Value       string
	Owner       string
	Read        bool
	Write       bool
	ChangeOwner bool
}

type Verb struct {
	Value   string
	Owner   string
	Read    bool
	Write   bool
	Execute bool
	Debug   bool
}

type Object struct {
	Properties map[string]Property
	Verbs      map[string]Verb
	Id         int
}

func breakSentence(s string) (string, string, string, string) {
	x, _ := shellwords.Parse(s)
	log.Printf("Parsed line into %v", x)
	x = append(x, "oops")
	x = append(x, "oops")
	x = append(x, "oops")
	x = append(x, "oops")
	return x[0], x[1], x[2], x[3]

}
func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
func SaveObject(o Object) {

	os.Mkdir("objects", 0777)
	txt, err := json.Marshal(o)
	panicErr(err)
	err = ioutil.WriteFile(fmt.Sprintf("objects/%v.json", o.Id), txt, 0600)
	panicErr(err)
}

func LoadObject(id string) Object {
	log.Println("Loading " + "objects/" + id + ".json")
	file, _ := ioutil.ReadFile("objects/" + id + ".json")

	data := Object{}

	_ = json.Unmarshal([]byte(file), &data)
	return data
}
func ParseDo(s string) (string, string) {
	log.Println("Splitting", s, "on", ".")
	ss := strings.Split(s, ".")
	ss = append(ss, "no property")
	return ss[0], ss[1]
}
func main() {
	coreObj := Object{}
	coreObj.Verbs = map[string]Verb{}
	coreObj.Properties = map[string]Property{}
	ver := Verb{Value: `
	var objStr, propStr string
	objStr, propStr = ParseDo(do)
	var o Object
	var p Property
	o= LoadObject(obj)
	p= Property{Value: ind}
	o.Properties[prop] = p
	SaveObject(obj)
	log.Println(obj)
	`}
	coreObj.Verbs["property"] = ver
	SaveObject(coreObj)
	i := interp.New(interp.Options{GoPath: build.Default.GOPATH, BuildTags: strings.Split("", ",")})
	if err := i.Use(stdlib.Symbols); err != nil {
		panic(err)
	}

	if err := i.Use(interp.Symbols); err != nil {
		panic(err)
	}
	if err := i.Use(interp.Symbols()); err != nil {
		panic(err)
	}
	/*
		if err := i.Use(syscall.Symbols); err != nil {
			panic(err)
		}

		// Using a environment var allows a nested interpreter to import the syscall package.
		if err := os.Setenv("YAEGI_SYSCALL", "1"); err != nil {
			panic(err)
		}

		if err := i.Use(unsafe.Symbols); err != nil {
			panic(err)
		}
		if err := os.Setenv("YAEGI_UNSAFE", "1"); err != nil {
			panic(err)
		}

		// Use of unrestricted symbols should always follow stdlib and syscall symbols, to update them.
		if err := i.Use(unrestricted.Symbols); err != nil {
			panic(err)
		}
	*/
	i.Eval(lib())
	log.Println("libs loaded")
	reader := bufio.NewReader(os.Stdin)
	var verb Verb
	for {
		fmt.Println(verb)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
		verb, directObject, preposition, indirectObject := breakSentence(text)
		call := lib() + ` 
		func runme() {
		do := "` + directObject + `"` + `
		prep := "` + preposition + `"` + `
		ind := "` + indirectObject + `"
		`
		obj, _ := ParseDo(directObject)
		do := LoadObject(obj)
		ve := do.Verbs[verb]
		log.Println(do)
		log.Println("Verb", verb, "is", ve)
		call = call + ve.Value + "}"
		log.Println("Evalling ", call)
		var v reflect.Value
		v, err := i.Eval(call)
		fmt.Println(err)
		if v.IsValid() {
			fmt.Println(v)
		}
	}
}

func lib() string {
	return `
	import (
	
		"os"

	)
	type Property struct {
		Value       string
		Owner       string
		Read        bool
		Write       bool
		ChangeOwner bool
	}
	
	type Verb struct {
		Value   string
		Owner   string
		Read    bool
		Write   bool
		Execute bool
		Debug   bool
	}
	
	type Object struct {
		Properties map[string]Property
		Verbs      map[string]Verb
		Id         int
	}
	
	func SaveObject(o Object) {
os.Mkdir("objects", 0777)
txt, err := json.Marshal(o)
panicErr(err)
err = ioutil.WriteFile(fmt.Sprintf("objects/%v.json", o.Id), txt, 0600)
panicErr(err)
}

func LoadObject(id string) Object {
log.Println("Loading " + "objects/" + id + ".json")
file, _ := ioutil.ReadFile("objects/" + id + ".json")

data := Object{}

_ = json.Unmarshal([]byte(file), &data)
return data
}
func ParseDo(s string) (string, string) {
log.Println("Splitting", s, "on", ".")
ss := strings.Split(s, ".")
ss = append(ss, "no property")
return ss[0], ss[1]
}`
}

/*
name
a string, the usual name for this object
owner
an object, the player who controls access to it
location
an object, where the object is in virtual reality
contents
a list of objects, the inverse of location
programmer
a bit, does the object have programmer rights?
wizard
a bit, does the object have wizard rights?
r
a bit, is the object publicly readable?
w
a bit, is the object publicly writable?
f
a bit, is the object fertile?

*/
