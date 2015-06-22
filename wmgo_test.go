package wmgo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

// ref 用于1:1 or 1:n, link用于m:n, 表名以类名小写为值,列名以json字段为值
type User struct {
	Id       string   `json:"id" bson:"_id" link:"usergroup"`
	Password string   `json:"-" bson:"password"`
	Name     string   `json:"nam" bson:"name"`
	RoleId   string   `json:"roleId" bson:"roleId" ref:"role._id"`
	TagsId   []string `json:"tagIds" bson:"tagIds" ref:"tag.name"`
}

type Role struct {
	Id   string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`
}

type Tag struct {
	Id   string `json:"-" bson:"_id"`
	Name string `json:"name" bson:"name"`
}

type Group struct {
	Id   string `json:"id" bson:"_id" link:"usergroup"`
	Name string `json:"name" bson:"name"`
}

type UserGroup struct {
	Id  string `json:"id" bson:"_id"`
	Uid string `json:"uid" bson:"uid" foreign:"user._id"`
	Gid string `json:"gid" bson:"gid" foreign:"group._id"`
}

var e *Engine = nil

func TestMain(m *testing.M) {
	e = NewEngine("", "test")
	e.db.DropDatabase()

	e.Map(User{})
	e.Map(Role{})
	e.Map(Tag{})
	e.Map(Group{})
	e.Map(UserGroup{})
	fmt.Printf("%s\n", e.Relations())

	m.Run()
}

func TestEmbed(t *testing.T) {
	id := bson.NewObjectId().Hex()
	e.Insert(&Role{Id: id, Name: "role"})

	// insert tags
	e.Insert(&Tag{Id: bson.NewObjectId().Hex(), Name: "tag1"})
	e.Insert(&Tag{Id: bson.NewObjectId().Hex(), Name: "tag2"})

	// insert groups
	gid1 := bson.NewObjectId().Hex()
	gid2 := bson.NewObjectId().Hex()
	e.Insert(&Group{Id: gid1, Name: "group1"})
	e.Insert(&Group{Id: gid2, Name: "group2"})

	// insert user
	uid := bson.NewObjectId().Hex()
	e.Insert(&User{Password: "1223", Id: uid, Name: "user", RoleId: id, TagsId: []string{"tag1", "tag2"}})

	// insert usergroup
	e.Insert(&UserGroup{Id: bson.NewObjectId().Hex(), Gid: gid1, Uid: uid})
	e.Insert(&UserGroup{Id: bson.NewObjectId().Hex(), Gid: gid2, Uid: uid})

	// 1:1
	q := Query{
		Embeds: []string{"role"},
	}
	rets, err := e.IntelligentQuery(q, &User{})

	assert.Nil(t, err)
	assert.Equal(t, len(rets), 1)
	assert.Equal(t, rets[0]["role"].(*Role).Name, "role")

	// 1:n
	q = Query{
		Embeds: []string{"tag"},
	}
	rets, err = e.IntelligentQuery(q, &User{})

	assert.Equal(t, rets[0]["tag"].([]Tag)[0].Name, "tag1")
	assert.Equal(t, rets[0]["tag"].([]Tag)[1].Name, "tag2")

	// m:n
	q = Query{
		Embeds: []string{"group"},
	}
	rets, err = e.IntelligentQuery(q, &User{})

	fmt.Printf("%s\n", rets)
	//assert.Equal(t, rets[0]["group"].([]Group)[0].Name, "group1")
	//assert.Equal(t, rets[0]["group"].([]Group)[1].Name, "group2")

}
