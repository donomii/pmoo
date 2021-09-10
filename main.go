package pmoo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

func BreakSentence(s string) (string, string, string, string) {
	x, _ := LexLine(s)
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
func FormatObject(id string) string {
	txt, err := json.MarshalIndent(LoadObject(id), "", " ")
	panicErr(err)
	return string(txt)
}
func DumpObject(id string) {
	fmt.Println(FormatObject(id))
}
func L(s interface{}) {
	log.Println(s)
}
func SaveObject(o Object) {

	os.Mkdir("objects", 0777)
	txt, err := json.MarshalIndent(o, "", " ")
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

func GetProperty(o Object, name string) *Property {
	val, ok := o.Properties[name]
	if ok {
		return &val
	}
	parent := o.Properties["parent"].Value
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetProperty(LoadObject(parent), name)
}

func GetVerb(o Object, name string) *Verb {
	val, ok := o.Verbs[name]
	if ok {
		return &val
	}
	parent := o.Verbs["parent"].Value
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetVerb(LoadObject(parent), name)
}

//from https://github.com/laurent22/massren/
func LexLine(editorCmd string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	for i := 0; i < len(editorCmd); i++ {
		c := editorCmd[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", editorCmd))
	}

	if current != "" {
		args = append(args, current)
	}

	if len(args) <= 0 {
		return []string{}, errors.New("Empty command line")
	}

	if len(args) == 1 {
		return args, nil
	}

	return args, nil
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
