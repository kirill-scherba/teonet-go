// Copyright 2019 Teonet-go authors.  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package teocdb (teo-cdb) is the Teonet database service package
//
// Install this go package:
//   go get github.com/kirill-scherba/teonet-go/services/teocdb
//
// Data base organisation
//
// This service uses ScyllaDB. If you install this service manually you need
// install ScyllaDB. Run Scylla in Docker:
//
//   https://www.scylladb.com/download/open-source/#docker
//
// Before you execute application which used this package you need install
// database schemas. Launch `cqlsh`:
//   docker exec -it scylla cqlsh
// and execute next commands:
/*
	create keyspace teocdb with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 };
	create table teocdb.map(key text, data blob, PRIMARY KEY(key));
*/
//
package teocdb

import (
	"fmt"

	"github.com/gocql/gocql"
	cdb "github.com/kirill-scherba/teonet-go/services/teocdb/teocdbcli"
)

// Process receiver to process teocdb commands
type Process struct{ tcdb *Teocdb }

// Teocdb is teocdb packet receiver
type Teocdb struct {
	session *gocql.Session
	process Process
	con     cdb.TeoConnector
}

// Connect to the cql cluster and return teocdb receiver
func Connect(con cdb.TeoConnector, hosts ...string) (tcdb *Teocdb, err error) {
	tcdb = &Teocdb{con: con}
	tcdb.process.tcdb = tcdb
	cluster := gocql.NewCluster(func() (h []string) {
		if h = hosts; len(h) == 0 {
			h = []string{"172.17.0.2", "172.17.0.3", "172.17.0.4"}
		}
		return
	}()...)
	cluster.Keyspace = "teocdb"
	cluster.Consistency = gocql.Quorum
	tcdb.session, _ = cluster.CreateSession()

	// Create keyspace and table
	const mapSchema = `
		// create KEYSPACE IF NOT EXISTS teocdb WITH replication = {
		// 	'class' : 'SimpleStrategy',
		// 	'replication_factor' : 3
		// };
		create TABLE IF NOT EXISTS teocdb.map(
			key text,
			data blob,
			PRIMARY KEY(key)
		)`
	if err = tcdb.execStmt(tcdb.session, mapSchema); err != nil {
		//t.Fatal("create table:", err)
	}
	return
}

// ExecStmt executes a statement string.
func (tcdb *Teocdb) execStmt(s *gocql.Session, stmt string) error {
	q := s.Query(stmt).RetryPolicy(nil)
	defer q.Release()
	return q.Exec()
}

// Close teocdb connection
func (tcdb *Teocdb) Close() {
	tcdb.session.Close()
}

// Update key value
func (tcdb *Teocdb) Update(key string, value []byte) (err error) {
	if err = tcdb.session.Query(`UPDATE map SET data = ? WHERE key = ?`,
		value, key).Exec(); err != nil {
		fmt.Printf("Insert Error: %s\n", err.Error())
	}
	return
}

// Get value by key
func (tcdb *Teocdb) Get(key string) (data []byte, err error) {
	if err := tcdb.session.Query(`SELECT data FROM map WHERE key = ? LIMIT 1`,
		key).Consistency(gocql.One).Scan(&data); err != nil {
		fmt.Printf("Get Error: %s\n", err.Error())
	}
	return
}

// List read and return array of all keys connected to selected key
func (tcdb *Teocdb) List(key string) (keyList cdb.KeyList, err error) {
	var keyOut string
	iter := tcdb.session.Query(`
		SELECT key FROM map WHERE key >= ? and key < ?
		ALLOW FILTERING`,
		key, key+"a").Iter()
	for iter.Scan(&keyOut) {
		fmt.Println("key:", keyOut)
		keyList.Append(keyOut)
	}
	return
}

// Process return command processing receiver
func (tcdb *Teocdb) Process() *Process {
	return &tcdb.process
}

// CmdBinary process CmdBinary command
func (proc *Process) CmdBinary(from string, cmd byte, data []byte) (err error) {
	fmt.Printf("Got CmdBinary: \n")
	var request, responce cdb.BinaryData
	err = request.UnmarshalBinary(data)
	if err != nil {
		fmt.Printf("Unmarshal Error: %s\n", err.Error())
		return
	}
	fmt.Println(request.Cmd, request.Key, request.Value)
	responce = request
	switch request.Cmd {
	case cdb.CmdSet:
		if err = proc.tcdb.Update(request.Key, request.Value); err != nil {
			fmt.Printf("Update Error: %s\n", err.Error())
			return
		}
		responce.Value = nil
	case cdb.CmdGet:
		if responce.Value, err = proc.tcdb.Get(request.Key); err != nil {
			fmt.Printf("Get Error: %s\n", err.Error())
			return
		}
	case cdb.CmdList:
		var keys cdb.KeyList
		if keys, err = proc.tcdb.List(request.Key); err != nil {
			fmt.Printf("Get Error: %s\n", err.Error())
			return
		}
		responce.Value, _ = keys.MarshalBinary()
	}
	var retdata []byte
	if retdata, err = responce.MarshalBinary(); err != nil {
		return
	}
	_, err = proc.tcdb.con.SendTo(from, cmd, retdata)
	return
}
