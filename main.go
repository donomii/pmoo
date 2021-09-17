package pmoo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/syscall"
	"github.com/traefik/yaegi/stdlib/unrestricted"
	"github.com/traefik/yaegi/stdlib/unsafe"
	etcd "go.etcd.io/etcd/client/v3"
)

var Cluster bool

var EtcdServers []string //= []string{"localhost:2379"}

type Message struct {
	Player                                                    string
	This                                                      string
	Verb                                                      string
	Dobj, Dpropstr, Prepstr, Iobj, Ipropstr, Dobjstr, Iobjstr string
	Data                                                      string
	From                                                      string
	Trace                                                     string
}

var Q chan *Message

func SetQ(queue chan *Message) {
	Q = queue
	log.Printf(" set queue to %v", Q)

}

func InputMsg(from, target, verb, arg1 string) {
	m := &Message{Player: from, This: target, Verb: verb, Data: arg1}
	log.Printf("**********Queueing Message %v to %v\n", m, Q)
	Q <- m
	log.Println("**************Message queued")
}

func Msg(from, target, verb, dobjstr, prep, iobjstr string) {
	m := &Message{Player: from, This: target, Verb: verb, Dobjstr: dobjstr, Prepstr: prep, Iobjstr: iobjstr}
	Q <- m
	log.Printf("Audit: Message %v\n", m)
}

func RawMsg(m Message) {
	Q <- &m
	log.Printf("Audit: Message %v\n", m)
}

type Property struct {
	Value       string
	Owner       string
	Read        bool
	Write       bool
	Execute     bool
	Debug       bool
	ChangeOwner bool
	Verb        bool
	Throff      bool
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

	locId := GetPropertyStruct(player, "location", 10).Value
	loc := LoadObject(locId)
	contains := SplitStringList(GetPropertyStruct(loc, "contains", 10).Value)
	contains = append([]string{locId}, contains...)
	for _, objId := range contains {
		out = append(out, objId)
	}
	return out
}

func FormatObject(id string) string {
	o := LoadObject(id)
	var out string
	loc := GetProp(id, "location")
	out = out + fmt.Sprintf("\n\nObject %v, %v at %v\nVerbs\n-----\n", id, GetProp(ToStr(o.Id), "name"), GetProp(loc, "name"))
	o = LoadObject(id)
	var verbs, props []string
	for k, v := range o.Properties {
		if v.Verb {
			verbs = append(verbs, k)
		} else {
			props = append(props, k)
		}

	}
	out = out + strings.Join(verbs, ",") + "\n\nProperties\n----------\n" + strings.Join(props, ",") + "\n"
	out = out + "Definitions\n-----------\n"
	for k, v := range o.Properties {
		out = out + fmt.Sprintf("%v: %v\n", k, v)
	}
	return out
}

func DumpObject(id string) {
	fmt.Println(FormatObject(id))
}
func L(s interface{}) {
	log.Println(s)
}
func SaveObject(o *Object) {
	txt, err := json.MarshalIndent(o, "", " ")
	panicErr(err)
	if Cluster {
		cli, err := etcd.New(etcd.Config{
			Endpoints:   EtcdServers,
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			log.Println("ERROR while storing key", o.Id)

			panic(err)
		}
		defer cli.Close()
		//var resp *etcd.PutResponse
		log.Printf("Storing key %v\n", ToStr(o.Id))
		_, err = cli.Put(context.TODO(), ToStr(o.Id), string(txt))
		if err != nil {
			log.Println("ERROR while storing key", o.Id)
			//log.Println(resp)
			log.Println(err)
		}

	} else {
		os.Mkdir("objects", 0777)

		err = ioutil.WriteFile(fmt.Sprintf("objects/%v.json", o.Id), txt, 0600)
		panicErr(err)
	}

}

func LoadObject(id string) *Object {
	if Cluster {
		var resp *etcd.GetResponse
		cli, err := etcd.New(etcd.Config{
			Endpoints:   EtcdServers,
			DialTimeout: 5 * time.Second,
		})
		resp, err = cli.Get(context.TODO(), id)
		if err != nil {
			log.Println(err)
			return nil
		}
		defer cli.Close()

		for _, ev := range resp.Kvs {
			//log.Printf("%s : %s\n", ev.Key, ev.Value)

			data := Object{}

			err = json.Unmarshal([]byte(ev.Value), &data)
			if err != nil {
				return nil
			}

			return &data
		}
		return nil
	} else {
		n_id, _ := strconv.Atoi(id)
		id = ToStr(n_id)
		//log.Println("Loading " + "objects/" + id + ".json")
		file, err := ioutil.ReadFile("objects/" + id + ".json")
		if err != nil {
			return nil
		}

		data := Object{}

		err = json.Unmarshal([]byte(file), &data)
		if err != nil {
			return nil
		}

		//fmt.Printf("%+v\n", data)
		return &data
	}
}

//Fixme copied
func GetPropertyStruct(o *Object, name string, timeout int) *Property {
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
		panic(fmt.Sprintf("No parent for %v when looking for property %v in %v", o.Id, name, o.Id))
	}
	parent := parentProp.Value
	//log.Printf("Searching %v, then parent %v\n", fmt.Sprintf("%v", o.Id), parent)
	//Object points to itself, time to quit
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetPropertyStruct(LoadObject(parent), name, timeout-1)
}

func GetProp(objstr, name string) string {
	p := GetPropertyStruct(LoadObject(objstr), name, 10)
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
		panic(fmt.Sprintf("No parent for %v when looking for verb %v in %v", o.Id, name, o.Id))
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

	//log.Printf("Searching %v, then parent %v\n", fmt.Sprintf("%v", o.Id), parent)
	//Object points to itself, time to quit
	if parent == fmt.Sprintf("%v", o.Id) {
		return nil
	}
	return GetVerbStruct(LoadObject(parent), name, timeout-1)
}

func GetFreshId() int {
	root := LoadObject("1")
	lastId := GetPropertyStruct(root, "lastId", 10)
	if lastId == nil {
		panic("Can't get lastId")
	}

	id, _ := strconv.Atoi(lastId.Value)
	id = id + 1
	newLastId := fmt.Sprintf("%v", id)
	lastId.Value = newLastId
	root.Properties["lastId"] = *lastId
	SaveObject(root)
	return id
}

//Copy a loaded MOO object.  Saves the copy before returning
func CloneObject(o *Object) *Object {
	out := *o
	name := out.Properties["name"]
	name.Value = "Copy of " + name.Value
	out.Properties["name"] = name

	out.Id = GetFreshId()
	SaveObject(&out)
	log.Println("Audit: Cloned ", o.Id)
	return &out
}

//Create a copy of a MOO object
func Clone(objstr string) string {
	sourceO := LoadObject(objstr)
	newO := CloneObject(sourceO)
	return ToStr(newO.Id)
}

//Set a property on a loaded object
func SetProperty(o *Object, name, value string) {
	log.Printf("Setting property %v on object %v to %v\n", name, o.Id, value)
	prop := GetPropertyStruct(o, name, 10)
	if prop == nil {
		prop = &Property{}
	}
	p := *prop
	p.Value = value
	o.Properties[name] = p
	SaveObject(o)
}

//Set a property on an object
func SetProp(objstr, name, value string) {
	log.Printf("Setting prop %v on object %v to %v\n", name, objstr, value)
	o := LoadObject(objstr)
	SetProperty(o, name, value)
}

func SetVerbStruct(o *Object, name, value string) {
	prop := GetVerbStruct(o, name, 10)
	if prop == nil {
		prop = &Property{Verb: true}
	}
	prop.Verb = true
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

func NewInterpreter() *interp.Interpreter {
	i := interp.New(interp.Options{GoPath: build.Default.GOPATH, BuildTags: strings.Split("", ",")})
	if err := i.Use(stdlib.Symbols); err != nil {
		panic(err)
	}

	if err := i.Use(interp.Symbols); err != nil {
		panic(err)
	}

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

	return i

}

func Eval(i *interp.Interpreter, code string) {
	prog := `
		
	func runme(){

	` + code + `
}
func main() {runme()}`
	log.Println("Evalling:", prog)

	_, err := i.Eval(prog)
	if err != nil {
		fmt.Println("An error occurred:", err)
	}
}

//Implementing move as a built-in because it is complicated enough to need it
func MoveObj(objstr, targetstr string) {
	log.Printf("Moving %v to %v", objstr, targetstr)
	//Remove from old location
	oldlocationstr := GetProp(objstr, "location")
	log.Printf("Old location: %v", oldlocationstr)
	oldloc := LoadObject(oldlocationstr)
	log.Printf("Old location object: %v", oldloc)
	oldcontainerstr := GetProp(oldlocationstr, "contains")
	newcontainerstr := RemoveFromStringList(oldcontainerstr, objstr)
	log.Printf("New container string %v", newcontainerstr)
	SetProp(oldlocationstr, "contains", newcontainerstr)

	//Add to target container
	oldtargetcontainerstr := GetProp(targetstr, "contains")
	newtargetcontainerstr := AddToStringList(oldtargetcontainerstr, objstr)
	log.Printf("xtainer string %v", newtargetcontainerstr)
	SetProp(targetstr, "contains", newtargetcontainerstr)

	//Set the object's new location
	SetProp(objstr, "location", targetstr)

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

func BuildDefinitions(player, this, verb, dobj, dpropstr, prepositionStr, iobj, ipropstr, dobjstr, iobjstr string) string {
	definitions := ` 

	player := LoadObject("` + player + `")  //an object, the player who typed the command` + `
	playerid := "` + player + `"  //an object id, the player who typed the command` + `
	this := "` + this + `" //an object, the object on which this verb was found
	caller := LoadObject("` + player + `")  //an object, the same as player
	verb := "` + verb + `" //a string, the first word of the command
	//argstr //a string, everything after the first word of the command
	//args a list of strings, the words in argstr
	dobjstr := "` + dobjstr + `"` + ` //a string, the direct object string found during parsing
	dpropstr := "` + dpropstr + `"` + ` //a string, the direct object property string found during parsing
	dobj := LoadObject("` + dobj + `")  //an object, the direct object value found during matching
	prepstr:= "` + prepositionStr + `" //a string, the prepositional phrase found during parsing
	iobjstr := "` + iobjstr + `" //a string, the indirect object string
	ipropstr := "` + ipropstr + `" //a string, the indirect object string
	iobj := LoadObject("` + iobj + `")  //an object, the indirect object value

`
	return definitions
}

func VerbSearch(o *Object, aName string) (*Object, *Property) {

	locId := GetPropertyStruct(o, "location", 10).Value
	loc := LoadObject(locId)
	roomContents := SplitStringList(GetPropertyStruct(loc, "contains", 10).Value)
	playerContents := SplitStringList(GetPropertyStruct(o, "contains", 10).Value)
	contains := append([]string{ToStr(o.Id), locId}, roomContents...)
	contains = append(contains, playerContents...)
	for _, objId := range contains {
		obj := LoadObject(objId)
		nameProp := GetVerbStruct(obj, aName, 10)
		if nameProp != nil {
			return obj, nameProp
		}
	}

	log.Printf("Failed to find verb %v here!\n", aName)
	return nil, nil

}

func NameSearch(o *Object, aName string) (*Object, *Property) {

	locId := GetPropertyStruct(o, "location", 10).Value
	loc := LoadObject(locId)
	contains := SplitStringList(GetPropertyStruct(loc, "contains", 10).Value)
	contains = append([]string{locId, ToStr(o.Id)}, contains...)
	for _, objId := range contains {
		obj := LoadObject(objId)
		nameProp := GetPropertyStruct(obj, "name", 10)
		if nameProp != nil {
			if nameProp.Value == aName {
				return obj, nameProp
			}
		}
	}

	log.Printf("Failed to find object %v here!\n", aName)
	return nil, nil

}

func ParseDo(s string, objId string) (string, string) {
	if s == "me" {
		return objId, ""
	}
	if s == "here" {
		return GetPropertyStruct(LoadObject(objId), "location", 10).Value, ""
	}

	//It's a generic object, look for it in the properties of #1
	if len(s) > 0 && s[0] == '$' {
		prop := s[1:]
		one := LoadObject("1")
		oneprop := GetPropertyStruct(one, prop, 10)
		if oneprop == nil {
			fmt.Printf("Could not find special property %v on 1\n", prop)
		}
		objstr := oneprop.Value
		return objstr, ""
	}

	//It's an object number
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
		log.Println("Splitting", s, "on", ".")
		ss := strings.Split(s, ".")
		ss = append(ss, "no property")
		if ss[0] == "me" {
			return objId, ss[1]
		}
		if ss[0] == "here" {
			return GetPropertyStruct(LoadObject(objId), "location", 10).Value, ss[1]
		}
		return ss[0], ss[1]
	}

	//It might be the name of an object somewhere close
	foundObjId, _ := NameSearch(LoadObject(objId), s)
	if foundObjId != nil {
		return ToStr(foundObjId.Id), ""
	}

	return "", ""
}
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func FindPreposition(words []string) int {
	preps := strings.Split("named that is with using at to in front of in inside into on top of on onto upon out of from inside from over through under underneath beneath behind beside for about is as off off of", " ")
	for _, p := range preps {
		//log.Println("Searching for ", p, "in", words)
		r := Find(words, p)
		if r > -1 {
			return r
		}
	}
	return -1
}

func BreakSentence(s string) (string, string, string, string) {
	x, _ := LexLine(s)
	log.Printf("Parsed line into %v", strings.Join(x, ":"))
	if len(x) == 0 {
		return "", "", "", ""
	}
	if len(x) == 1 {
		return s, "", "", ""
	}
	if len(x) == 2 {
		return x[0], x[1], "", ""
	}
	verb := x[0]
	x = x[1:]
	preppos := FindPreposition(x)
	if preppos < 0 {
		return verb, strings.Join(x, " "), "", ""
	}

	do := x[:preppos]
	prep := x[preppos]
	io := x[preppos+1:]

	return verb, strings.Join(do, " "), prep, strings.Join(io, " ")

}
func SetThroffVerb(obj, name, code string) {
	log.Println("SetThroffVerb: ", obj, name, code)
	o := LoadObject(obj)
	v := Property{Value: code, Verb: true, Throff: true}
	log.Println("object:", o)
	o.Properties[name] = v
	SaveObject(o)
}
