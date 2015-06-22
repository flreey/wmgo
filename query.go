package wmgo

import (
	"fmt"
	"strings"
)
import "reflect"

type M map[string]interface{}

type Query struct {
	Ids      []string //ids + filter as find() params
	Sort     []string
	Selector M
	Start    int
	Limit    int //max 100
	Filter   M
	Embeds   []string
	Includes []string //won't use in IntelligentQuery
}

func NewQuery() *Query {
	return &Query{
		make([]string, 0),
		make([]string, 0),
		M{},
		0,
		100,
		M{},
		make([]string, 0),
		make([]string, 0),
	}
}

func (self *Query) Finder() M {
	q := self.Filter
	if len(self.Ids) > 0 {
		q = M{"_id": M{"$in": self.Ids}}
	}
	return q
}

func (self Query) TrimSelector(obj interface{}) M {
	v := reflect.ValueOf(obj).Elem().Type()
	n := v.NumField()
	for i := 0; i < n; i++ {
		field := v.Field(i)
		if tag := field.Tag.Get("json"); tag == "-" {
			delete(self.Selector, strings.ToLower(field.Name))
		}
	}
	return self.Selector
}

func (self *Query) String() string {
	return fmt.Sprintf("finder: %s, sort: %s, selector: %s, start %s, limit: %s, Embeds: %s, Includes: %s",
		self.Finder(), self.Sort, self.Selector, self.Start, self.Limit, self.Embeds, self.Includes)
}
