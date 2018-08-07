package main

import (
	"fmt"
	"mint/jsonstore"
	"os"

	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	mgo "gopkg.in/mgo.v2"

	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	defaultMongoHost = "localhost"
)

func main() {
	initJSONStore()
}

func initJSONStore() error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create the application
	var app types.Application

	mongoHost := os.Args[2]
	if mongoHost == "" {
		mongoHost = defaultMongoHost
	}
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		panic(err)
	}
	db := session.DB("tendermintdb")

	// Clean the DB on each reboot
	collections := [7]string{"posts", "comments", "users", "userpostvotes", "usercommentvotes", "validators", "uservalidatorvotes"}

	for _, collection := range collections {
		db.C(collection).RemoveAll(nil)
	}

	app = jsonstore.NewJSONStoreApplication(db)

	// Start the listener
	proxyAppPort := os.Args[1]
	srv, err := server.NewServer(fmt.Sprintf("tcp://0.0.0.0:%s", proxyAppPort), "socket", app)
	if err != nil {
		return err
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		return err
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return nil
}
