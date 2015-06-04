package wmgo

import (
	"reflect"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Engine struct {
	host string
	db   *mgo.Database
}

func NewEngine(host string, dbname string) *Engine {
	s, _ := mgo.Dial("")
	db := s.DB(dbname)
	return &Engine{host, db}
}

func (self *Engine) Register(obj interface{}) interface{} {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		panic("should be ptr")
	}

	v := reflect.ValueOf(obj).Elem()
	if _, ok := reflect.TypeOf(obj).Elem().FieldByName("Id"); ok {
		v.FieldByName("Id").Set(reflect.ValueOf(bson.NewObjectId().Hex()))
	}

	coll := self.db.C(strings.ToLower(reflect.TypeOf(obj).Elem().Name()))
	v.FieldByName("context").Set(reflect.ValueOf(&context{coll, obj}))
	return obj
}

type context struct {
	coll *mgo.Collection
	_v   interface{}
}

func (self *context) Insert() error {
	return self.coll.Insert(self._v)
}

func (self *context) Upsert(q interface{}) error {
	_, err := self.coll.Upsert(q, self._v)
	return err
}

func (self *context) Find(q interface{}, all interface{}) (err error) {
	return self.coll.Find(q).All(all)
}
