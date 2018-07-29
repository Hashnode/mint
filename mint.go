package main

import (
	"fmt"
	"mint/jsonstore"
	"os"
	"strconv"

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

	fmt.Println("running aginst mongo: ", os.Getenv("MONGO_URL"))
	session, err := mgo.Dial(os.Getenv("MONGO_URL"))
	if err != nil {
		panic(err)
	}
	db := session.DB("tendermintdb")

	// Clean the DB on each reboot
	collections := [6]string{"posts", "comments", "users", "userpostvotes", "usercommentvotes", "validators"}
	// collections := [6]string{"posts", "comments", "users", "userpostvotes", "usercommentvotes"}

	for _, collection := range collections {
		db.C(collection).RemoveAll(nil)
	}

	app = jsonstore.NewJSONStoreApplication(db)

	// Start the listener
	port := 46658
	if os.Args[1] != "" {
		port, err = strconv.Atoi(os.Args[1])
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(os.Args[1:])
	fmt.Println(port)
	srv, err := server.NewServer(fmt.Sprintf("tcp://0.0.0.0:%d", port), "socket", app)
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
