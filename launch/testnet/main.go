package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/logging"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

const (
	// name (wihtout ext) for the config file to look for
	configFileName = "dusk"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Fprintln(os.Stdout, "initializing node...")
	// Loading all node configurations. Fail-fast if critical error occurs
	err := cfg.Load(configFileName, nil, nil)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	port := cfg.Get().Network.Port
	rand.Seed(time.Now().UnixNano())

	// Set up logging.
	// Any subsystem should be initialized after config and logger loading
	output := cfg.Get().Logger.Output
	var logFile *os.File
	if cfg.Get().Logger.Output != "stdout" {
		logFile, err = os.Create(output + port + ".log")
		if err != nil {
			log.Panic(err)
		}
		defer logFile.Close()
	} else {
		logFile = os.Stdout
	}

	logging.InitLog(logFile)

	log.Infof("Loaded config file %s", cfg.Get().UsedConfigFile)
	log.Infof("Selected network  %s", cfg.Get().General.Network)

	// Setting up the EventBus and the startup processes (like Chain and CommitteeStore)
	srv := Setup()
	defer srv.Close()

	// Setting up profiling tools, if enabled
	s := setupProfiles(srv.rpcBus)
	defer s.Close()

	//start the connection manager
	connMgr := NewConnMgr(CmgrConfig{
		Port:     port,
		OnAccept: srv.OnAccept,
		OnConn:   srv.OnConnection,
	})

	// fetch neighbours addresses from the Seeder
	ips := ConnectToSeeder()

	// trying to connect to the peers
	for _, ip := range ips {
		if err := connMgr.Connect(ip); err != nil {
			log.WithField("IP", ip).Warnln(err)
		}
	}

	fmt.Fprintln(os.Stdout, "initialization complete")

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt

	// Graceful shutdown of listening components
	msg := message.New(topics.Quit, bytes.Buffer{})
	srv.eventBus.Publish(topics.Quit, msg)

	log.WithField("prefix", "main").Info("Terminated")
}

func setupProfiles(r *rpcbus.RPCBus) *diagnostics.ProfileSet {

	s := diagnostics.NewProfileSet()
	profiles := cfg.Get().Profile
	// Expecting an array of profiles.
	// Add empty [[profile]] to enable the listener
	if len(profiles) > 0 {

		for _, i := range profiles {

			if len(i.Name) == 0 {
				continue
			}

			p := diagnostics.NewProfile(i.Name, i.Interval, i.Duration, i.Start)
			if err := s.Spawn(p); err != nil {
				log.Panicf("Profiling task error: %s", err.Error())
			}
		}

		go s.Listen(r)
	}

	return &s
}
