package main

import (
	"fmt"
	"testing"

	"gopkg.in/mgo.v2/bson"
)

type Model struct {
	*context `bson:"-"`
	Id       string `bson:"_id"`
	Name     string
}

func TestEngine(t *testing.T) {
	e := NewEngine("", "test")
	m := e.Register(new(Model)).(*Model)

	m.Name = "A"
	m.Insert()
	fmt.Printf("!%s\n", m.Id)
	m.Name = "B"
	err := m.Upsert(bson.M{"_id": m.Id})
	if err != nil {
		panic(err)
	}

	ms := make([]Model, 0)
	err = m.Find(nil, &ms)
	if err != nil {
		panic(err)
	}

	if len(ms) != 1 {
		panic("len(ms) should be 1")
	}
}
