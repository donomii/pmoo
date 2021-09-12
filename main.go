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

type Object struct {
	Properties map[string]Property
	Id         int
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func ToStr(i interface{}) string {
	return fmt.Sprintf("%v", i)
}

func VisibleObjects(player *Object) []string {
	out := []string{}

	locId := GetProperty(player, "location", 10).Value
	loc := LoadObject(locId)
	contains := SplitStringList(GetProperty(loc, "contains", 10).Value)
	contains = append([]string{locId}, contains...)
	for _, objId := range contains {
		out = append(out, objId)
	}
	return out
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

//Fixme copied
func GetProperty(o *Object, name string, timeout int) *Property {
	//log.Println(o)
	if timeout < 1 {
		log.Printf("Timeout while looking up %v on %v\n", name, o.Id)
		return nil
	}
	if o == nil {
		return nil
	}

	val, ok := o.Properties[name]
	if ok {
		if val.Verb {
			fmt.Printf("Can't find property '%v', but could find verb '%v'\n", name, name)
			return nil
		}
		return &val
	}
	parentProp, ok := o.Properties["parent"]
	if !ok {
		//All objects must have a parent
		panic(fmt.Sprintf("No parent for %v", o.Id))
	}
	parent := parentProp.Value
	log.Printf("Searching %v, then parent %v\n", fmt.Sprintf("%v", o.Id), parent)
	//Object points to itself, time to quit
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetProperty(LoadObject(parent), name, timeout-1)
}

func GetProp(objstr, name string) string {
	p := GetProperty(LoadObject(objstr), name, 10)
	if p != nil {
		return p.Value
	} else {
		return ""
	}
}

func GetVerb(objstr, name string) string {
	p := GetVerbStruct(LoadObject(objstr), name, 10)
	if p != nil {
		return p.Value
	} else {
		return ""
	}
}

func GetVerbStruct(o *Object, name string, timeout int) *Property {
	//log.Println(o)
	if timeout < 1 {
		log.Printf("Timeout while looking up %v on %v\n", name, o.Id)
		return nil
	}
	if o == nil {
		return nil
	}

	parentProp, ok := o.Properties["parent"]
	if !ok {
		//All objects must have a parent
		panic(fmt.Sprintf("No parent for %v", o.Id))
	}
	parent := parentProp.Value

	val, ok := o.Properties[name]
	if ok {
		if !val.Verb {
			//fmt.Printf("Can't find verb '%v', but could find property '%v'\n", name, name)
			//return nil
			return GetVerbStruct(LoadObject(parent), name, timeout-1)
		}
		return &val
	}

	log.Printf("Searching %v, then parent %v\n", fmt.Sprintf("%v", o.Id), parent)
	//Object points to itself, time to quit
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetVerbStruct(LoadObject(parent), name, timeout-1)
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

	name := out.Properties["name"]
	name.Value = "Copy of " + name.Value
	out.Properties["name"] = name

	out.Id = GetFreshId()
	SaveObject(&out)
	return &out
}

func Clone(objstr string) string {
	sourceO := LoadObject(objstr)
	newO := CloneObject(sourceO)
	return ToStr(newO.Id)
}

func SetProperty(o *Object, name, value string) {
	prop := GetProperty(o, name, 10)
	if prop == nil {
		prop = &Property{}
	}
	p := *prop
	p.Value = value
	o.Properties[name] = p
	SaveObject(o)
}
func SetProp(objstr, name, value string) {
	o := LoadObject(objstr)
	SetProperty(o, name, value)
}

func SetVerbStruct(o *Object, name, value string) {
	prop := GetVerbStruct(o, name, 10)
	if prop == nil {
		prop = &Property{Verb: true}
	}
	p := *prop
	p.Value = value
	o.Properties[name] = p
	SaveObject(o)
}

func SetVerb(objstr, name, value string) {
	o := LoadObject(objstr)
	SetVerbStruct(o, name, value)
}

func SplitStringList(s string) []string {
	return strings.Split(s, ",")
}

func BuildStringList(l []string) string {
	return strings.Join(l, ",")
}

func AddToStringList(l, s string) string {
	if l == "" {
		return s
	}
	return l + "," + s
}

func RemoveFromStringList(l, s string) string {
	list := SplitStringList(l)
	var out []string
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return BuildStringList(out)
}

//Implementing move as a built-in because it is complicated enough to need it
func MoveObj(objstr, targetstr string) {
	log.Printf("Moving %v to %v", objstr, targetstr)
	obj := LoadObject(objstr)
	target := LoadObject(targetstr)

	//Remove from old location
	oldlocationstr := GetProperty(obj, "location", 10).Value
	log.Printf("Old location: %v", oldlocationstr)
	oldloc := LoadObject(oldlocationstr)
	log.Printf("Old location object: %v", oldloc)
	oldcontainerstr := GetProperty(oldloc, "contains", 10).Value
	newcontainerstr := RemoveFromStringList(oldcontainerstr, objstr)
	log.Printf("New container string %v", newcontainerstr)
	SetProperty(oldloc, "contains", newcontainerstr)

	//Add to target container
	oldtargetcontainerstr := GetProperty(target, "contains", 10).Value
	newtargetcontainerstr := AddToStringList(oldtargetcontainerstr, objstr)
	log.Printf("New target container string %v", newtargetcontainerstr)
	SetProperty(target, "contains", newtargetcontainerstr)

	//Set the object's new location
	SetProperty(obj, "location", targetstr)

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
