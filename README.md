Wrap gopkg.in/mgo.v2

usage:

	e := NewEngine("", "test")
	m := e.Register(new(Model)).(*Model)
	m.Insert()

	q := bson.M{"_id": ""}
	m.Upsert(q)
