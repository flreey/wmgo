package wmgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/mgo.v2"
)

var ErrNotFoundEmbed = errors.New("not found embed")

type table struct {
	Type     reflect.Type
	Refs     map[string]Ref  // used for 1:1 or 1:n key is reference table name
	Links    map[string]Link // used for n:m, key is linked table name
	Foreigns map[string]Foreign
}

type Ref struct {
	IsArray   bool
	FieldName string // current table field name
	RefTag    string // ref table field bson tag
}

type Link struct {
	FieldName string // current table field name
}

type Foreign struct {
	FieldName string // current table field name
	FieldTag  string // current table field json tag
	RefTag    string // ref table field bson tag
}

type Engine struct {
	tables map[string]table
	db     *mgo.Database
}

func NewEngine(host string, dbname string) *Engine {
	s, _ := mgo.Dial("")
	db := s.DB(dbname)
	return &Engine{make(map[string]table, 0), db}
}

func (self *Engine) Map(obj interface{}) {
	value := reflect.ValueOf(obj)
	t := table{
		Type: reflect.TypeOf(obj),
	}

	refs := make(map[string]Ref, 0)
	links := make(map[string]Link, 0)
	foreigns := make(map[string]Foreign, 0)

	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		tag := field.Tag
		if tag.Get("ref") != "" {
			tagName := strings.Split(tag.Get("ref"), ".")
			tname, refTag := tagName[0], tagName[1]
			isArray := false
			if field.Type.Kind() == reflect.Slice {
				isArray = true
			}
			refs[tname] = Ref{isArray, field.Name, refTag}
			continue
		}

		if tag.Get("link") != "" {
			linkTable := tag.Get("link")
			links[linkTable] = Link{field.Name}
			continue
		}

		if tag.Get("foreign") != "" {
			tagName := strings.Split(tag.Get("foreign"), ".")
			tname, refTag := tagName[0], tagName[1]
			foreigns[tname] = Foreign{field.Name, tag.Get("json"), refTag}
			continue
		}
	}
	t.Refs = refs
	t.Links = links
	t.Foreigns = foreigns
	self.tables[getObjName(obj)] = t
}

func getObjName(obj interface{}) string {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return strings.ToLower(reflect.ValueOf(obj).Type().Name())
	}
	return strings.ToLower(reflect.ValueOf(obj).Elem().Type().Name())
}

func (self *Engine) Relations() string {
	bytes, _ := json.MarshalIndent(self.tables, "", "\t")
	return string(bytes)
}

func (self *Engine) getColl(obj interface{}) *mgo.Collection {
	collName := getObjName(obj)
	return copyCollection(self.db.C(collName))
}

func (self *Engine) IntelligentQuery(q Query, obj interface{}) ([]map[string]interface{}, error) {
	coll := self.getColl(obj)
	defer closeCollection(coll)

	iter := coll.Find(q.Finder()).Select(q.Selector).Sort(q.Sort...).Skip(q.Start).Limit(q.Limit).Iter()
	rets := make([]map[string]interface{}, 0)
	for {
		if !iter.Next(obj) {
			return rets, iter.Close()
		}

		m := struct2map(obj)

		for _, embed := range q.Embeds {
			embedValue, err := self.getEmbed(obj, embed)
			if err != nil {
				if err == mgo.ErrNotFound {
					continue
				}
				return rets, err
			}
			m[embed] = embedValue
		}
		rets = append(rets, m)
	}
	return rets, iter.Close()
}

func (self *Engine) getEmbed(obj interface{}, embed string) (interface{}, error) {
	table := self.tables[getObjName(obj)]
	if _, ok := table.Refs[embed]; ok {
		return self.one2Many(obj, embed)
	}

	linkTable := self.getLinkTable(getObjName(obj), embed)
	fmt.Printf("linkTable %s \n", linkTable)
	if linkTable == "" {
		return nil, ErrNotFoundEmbed
	}

	return self.many2Many(obj, linkTable, embed)
}

func (self *Engine) getLinkTable(table1 string, table2 string) string {
	for k, _ := range self.tables[table1].Links {
		if _, ok := self.tables[table2].Links[k]; ok {
			return k
		}
	}
	return ""
}

func (self *Engine) one2Many(obj interface{}, embed string) (interface{}, error) {
	table := self.tables[getObjName(obj)]
	ref := table.Refs[embed]
	embedTable := self.tables[embed]

	zeroV := reflect.New(embedTable.Type).Interface()
	coll := self.getColl(zeroV)
	defer closeCollection(coll)

	slicev := reflect.SliceOf(embedTable.Type)
	elemp := reflect.New(slicev)

	v := reflect.ValueOf(obj).Elem()
	q := M{ref.RefTag: v.FieldByName(ref.FieldName).String()}
	if ref.IsArray {
		strs := make([]string, 0)
		vs := v.FieldByName(ref.FieldName)
		for i := 0; i < vs.Len(); i++ {
			strs = append(strs, vs.Index(i).String())
		}
		q = M{ref.RefTag: M{"$in": strs}}
	}
	err := coll.Find(q).All(elemp.Interface())
	fmt.Printf("%s\n", q)
	if err != nil {
		panic(err)
	}

	if !ref.IsArray {
		return elemp.Elem().Index(0).Addr().Interface(), err
	}
	return elemp.Elem().Interface(), err
}

func (self *Engine) many2Many(obj interface{}, link string, embed string) (interface{}, error) {
	// get link values
	linkTable := self.tables[link]
	foreign := linkTable.Foreigns[getObjName(obj)]
	v := reflect.ValueOf(obj).Elem()
	value := ""
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("bson") == foreign.RefTag {
			value = v.Field(i).String()
			break
		}
	}
	q := M{foreign.FieldTag: value}
	elem, err := self.all(q, linkTable)
	if err != nil {
		return nil, err
	}

	elemp := reflect.ValueOf(elem)

	// get embed values
	ids := make([]interface{}, 0)
	embedForeign := linkTable.Foreigns[embed]

	for i := 0; i < elemp.Len(); i++ {
		ids = append(ids, elemp.Index(i).FieldByName(embedForeign.FieldName).Interface())
	}

	q = M{embedForeign.RefTag: M{"$in": ids}}
	embedTable := self.tables[embed]
	return self.all(q, embedTable)
}

func (self *Engine) all(q interface{}, t table) (interface{}, error) {
	zeroV := reflect.New(t.Type).Interface()
	embdeColl := self.getColl(zeroV)
	defer closeCollection(embdeColl)

	slicev := reflect.SliceOf(t.Type)
	elemp := reflect.New(slicev)
	err := embdeColl.Find(q).All(elemp.Interface())
	return elemp.Elem().Interface(), err
}

func struct2map(obj interface{}) map[string]interface{} {
	elem := reflect.ValueOf(obj).Elem()
	m := make(map[string]interface{}, 0)
	for i := 0; i < elem.NumField(); i++ {
		tag := elem.Type().Field(i).Tag
		if tag.Get("json") != "-" {
			m[tag.Get("json")] = elem.Field(i).Interface()
		}
	}
	return m
}

func (self *Engine) Insert(obj interface{}) error {
	coll := self.getColl(obj)
	defer closeCollection(coll)
	return coll.Insert(obj)
}
