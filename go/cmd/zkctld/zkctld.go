/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// zkctld is a daemon that starts or initializes ZooKeeper with Vitess-specific
// configuration. It will stay running as long as the underlying ZooKeeper
// server, and will pass along SIGTERM.
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"vitess.io/vitess/go/exit"
	"vitess.io/vitess/go/vt/log"
	"vitess.io/vitess/go/vt/logutil"
	"vitess.io/vitess/go/vt/zkctl"

	// Include deprecation warnings for soon-to-be-unsupported flag invocations.
	_flag "vitess.io/vitess/go/internal/flag"
)

var (
	zkCfg = flag.String("zk.cfg", "6@<hostname>:3801:3802:3803",
		"zkid@server1:leaderPort1:electionPort1:clientPort1,...)")
	myID = flag.Uint("zk.myid", 0,
		"which server do you want to be? only needed when running multiple instance on one box, otherwise myid is implied by hostname")
)

func main() {
	defer exit.Recover()
	defer logutil.Flush()

	_flag.Parse()

	zkConfig := zkctl.MakeZkConfigFromString(*zkCfg, uint32(*myID))
	zkd := zkctl.NewZkd(zkConfig)

	if zkd.Inited() {
		log.Infof("already initialized, starting without init...")
		if err := zkd.Start(); err != nil {
			log.Errorf("failed start: %v", err)
			exit.Return(255)
		}
	} else {
		log.Infof("initializing...")
		if err := zkd.Init(); err != nil {
			log.Errorf("failed init: %v", err)
			exit.Return(255)
		}
	}

	log.Infof("waiting for signal or server shutdown...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-zkd.Done():
		log.Infof("server shut down on its own")
	case <-sig:
		log.Infof("signal received, shutting down server")

		// Action to perform if there is an error
		if err := zkd.Shutdown(); err != nil {
			log.Errorf("error during shutdown:%v", err)
			exit.Return(1)
		}
	}
}
