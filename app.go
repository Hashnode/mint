package main

import (
	"os"
	"tendermint-app/jsonstore"

	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"
	mgo "gopkg.in/mgo.v2"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"
)

func main() {
	initJSONStore()
}

func initJSONStore() error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create the application
	var app types.Application

	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	db := session.DB("tendermintdb")

	errDrop := db.DropDatabase() // Clean the DB on each reboot

	if errDrop != nil {
		panic(errDrop)
	}

	app = jsonstore.NewJSONStoreApplication(db)

	// Start the listener
	srv, err := server.NewServer("tcp://0.0.0.0:46658", "socket", app)
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
