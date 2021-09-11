package pmoo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Property struct {
	Value       string
	Owner       string
	Read        bool
	Write       bool
	Execute     bool
	Debug       bool
	ChangeOwner bool
	Verb        bool
}

type Verb struct {
	Value       string
	Owner       string
	Read        bool
	Write       bool
	Execute     bool
	Debug       bool
	ChangeOwner bool
	Verb        bool
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
func SaveObject(o *Object) {

	os.Mkdir("objects", 0777)
	txt, err := json.MarshalIndent(o, "", " ")
	panicErr(err)
	err = ioutil.WriteFile(fmt.Sprintf("objects/%v.json", o.Id), txt, 0600)
	panicErr(err)
}

func LoadObject(id string) *Object {
	log.Println("Loading " + "objects/" + id + ".json")
	file, err := ioutil.ReadFile("objects/" + id + ".json")
	if err != nil {
		return nil
	}

	data := Object{}

	err = json.Unmarshal([]byte(file), &data)
	if err != nil {
		return nil
	}
	return &data
}
func ParseDo(s string, objId string) (string, string) {
	if s == "me" {
		return objId, ""
	}
	if s == "here" {
		return GetProperty(LoadObject(objId), "location", 10).Value, ""
	}
	if s[0] == '#' {
		s = s[1:]
		log.Println("Splitting", s, "on", ".")
		ss := strings.Split(s, ".")
		ss = append(ss, "no property")
		if ss[0] == "me" {
			return objId, ss[1]
		}
		if ss[0] == "here" {
			return GetProperty(LoadObject(objId), "location", 10).Value, ss[1]
		}
		return ss[0], ss[1]
	}
	return s, ""
}

func GetProperty(o *Object, name string, timeout int) *Property {
	if timeout < 1 {
		log.Printf("Timeout while looking up %v on %v\n", name, o.Id)
		return nil
	}
	if o == nil {
		return nil
	}
	val, ok := o.Properties[name]
	if ok {
		return &val
	}
	parent := o.Properties["parent"].Value
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetProperty(LoadObject(parent), name, timeout-1)
}

func GetVerb(o *Object, name string, timeout int) *Verb {
	//log.Println(o)
	if timeout < 1 {
		log.Printf("Timeout while looking up %v on %v\n", name, o.Id)
		return nil
	}
	if o == nil {
		return nil
	}

	val, ok := o.Verbs[name]
	if ok {
		return &val
	}
	parentProp, ok := o.Properties["parent"]
	if !ok {
		panic(fmt.Sprintf("No parent for %v", o.Id))
	}
	parent := parentProp.Value
	log.Printf("Searching %v, then parent %v\n", fmt.Sprintf("%v", o.Id), parent)
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetVerb(LoadObject(parent), name, timeout-1)
}

func GetFreshId() int {
	root := LoadObject("0")
	lastId := GetProperty(root, "lastId", 10)
	if lastId == nil {
		panic("Can't get lastId")
	}

	id, _ := strconv.Atoi(lastId.Value)
	id = id + 1
	newLastId := fmt.Sprintf("%v", id)
	lastId.Value = newLastId
	root.Properties["lastId"] = *lastId
	return id

}

func CloneObject(o *Object) *Object {
	out := *o
	desc := out.Properties["description"]
	desc.Value = "Copy of " + desc.Value
	out.Properties["description"] = desc
	out.Id = GetFreshId()
	return &out
}

func SetProperty(o *Object, name, value string) {
	prop := GetProperty(o, name, 10)
	if prop == nil {
		panic("Can't get property")
	}
	p := *prop
	p.Value = value
	o.Properties[name] = p
	SaveObject(o)
}

func SplitStringList(s string) []string {
	return strings.Split(s, ",")
}

func BuildStringList(l []string) string {
	return strings.Join(l, ",")
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
