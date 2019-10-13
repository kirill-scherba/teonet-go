// Copyright 2019 Teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Teousers db module.
//
// This service uses Scylla database and gocql and gocqlx packages to work with
// bd. Usually teousers package used in Teonet teocdb service application to
// provide users database functions to other teonet network applications.

package teousers

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/scylladb/gocqlx"
	"github.com/scylladb/gocqlx/table"
)

// User data structure
type User struct {
	UserID      gocql.UUID
	AccessToken gocql.UUID
	UserName    string
	AvatarID    gocql.UUID
	GravatarID  string
	Online      bool
	LastOnline  time.Time
}

// db data structure and methods receiver.
type db struct {
	*Users
	session      *gocql.Session
	usersTable   *table.Table
	userMetadata table.Metadata
}

// newDb creates new db structure.
func newDb(hosts ...string) (d *db, err error) {
	d = &db{
		userMetadata: table.Metadata{
			Name: "users",
			Columns: []string{
				"user_id",
				"access_token",
				"user_name",
				"avatar_id",
				"gravatar_id",
				"online",
				"last_online",
			},
			PartKey: []string{"user_id"},
			SortKey: []string{},
		},
	}
	// usersTable allows for simple CRUD operations based on personMetadata.
	d.usersTable = table.New(d.userMetadata)
	err = d.connect(hosts...)
	return
}

// connect to db.
func (d *db) connect(hosts ...string) (err error) {
	keyspace := "teousers"
	cluster := gocql.NewCluster(func() (h []string) {
		if h = hosts; len(h) > 0 {
			keyspace = h[0]
			h = h[1:]
		}
		if len(h) == 0 {
			h = []string{"172.17.0.2", "172.17.0.3", "172.17.0.4"}
		}
		return
	}()...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	d.session, err = cluster.CreateSession()
	return
}

// close db
func (d *db) close() {
	d.session.Close()
}

// set add new user or update existing. First input parameter is structure with
// filled UserID, and all other fields from User structure needs to set
// (usaly it may be User structure with all fields filled). Next parameters
// is column names which will be set to database, it may be ommited and than
// all columns sets.
func (d *db) set(u interface{}, columns ...string) (err error) {
	var stmt string
	var names []string
	if len(columns) == 0 {
		stmt, names = d.usersTable.Insert()
	} else {
		stmt, names = d.usersTable.Update(columns...)
	}
	q := gocqlx.Query(d.session.Query(stmt), names).BindStruct(u)
	fmt.Println(q.String())
	return q.ExecRelease()
}

// get returns select by primary key (UserID) statement. First input parameter
// is structure with filled UserID, and all other fields to reseive User structure
// (usaly it may be User structure). Next parameters is column names which need
// to get, it may be ommited and than all columns returns.
func (d *db) get(u interface{}, columns ...string) (err error) {
	stmt, names := d.usersTable.Get(columns...)
	q := gocqlx.Query(d.session.Query(stmt), names).BindStruct(u)
	fmt.Println(q.String())
	return q.GetRelease(u)
}

// delete record by user_id. Input parameter is structure with filled UserID
// field.
func (d *db) delete(u interface{}) (err error) {
	stmt, names := d.usersTable.Delete()
	q := gocqlx.Query(d.session.Query(stmt), names).BindStruct(u)
	fmt.Println(q.String())
	return q.ExecRelease()
}
