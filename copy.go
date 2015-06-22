package wmgo

import (
	"fmt"
	"strings"

	"github.com/qiniu/log"

	mgo "gopkg.in/mgo.v2"
)

const (
	copySessionMaxRetry = 5
)

func copySession(s *mgo.Session) *mgo.Session {
	for i := 0; i < copySessionMaxRetry; i++ {
		res := s.Copy()
		err := checkSession(res)
		if err == nil {
			return res
		}
		closeSession(res)
		log.Warn("[MGO2_COPY_SESSION] copy session and check failed:", err)
		if isServersFailed(err) {
			panic("[MGO2_COPY_SESSION_FAILED] servers failed")
		}
	}
	msg := fmt.Sprintf("[MGO2_COPY_SESSION_FAILED] failed after %d retries", copySessionMaxRetry)
	log.Error(msg)
	panic(msg)
}

func isServersFailed(err error) bool {
	return strings.Contains(err.Error(), "no reachable servers")
}

func checkSession(s *mgo.Session) (err error) {
	return s.Ping()
}

func copyDatabase(db *mgo.Database) *mgo.Database {
	return copySession(db.Session).DB(db.Name)
}

func closeSession(s *mgo.Session) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("[MGO2_CLOSE_SESSION_RECOVER] close session panic", err)
		}
	}()
	s.Close()
}

func closeCollection(coll *mgo.Collection) {
	closeSession(coll.Database.Session)
}

func copyCollection(coll *mgo.Collection) *mgo.Collection {
	return copyDatabase(coll.Database).C(coll.Name)
}
