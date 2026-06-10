package main

import (
	"log"
	"net"
	"strconv"

	"github.com/iomz/golemu/config"
	"github.com/iomz/golemu/connection"
	"github.com/iomz/golemu/server"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	cmd := kingpin.MustParse(config.App.Parse(os.Args[1:]))

	if *config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	var exitCode int
	switch cmd {
	case config.Client.FullCommand():
		client := connection.NewClient(config.IP.String(), *config.Port)
		exitCode = client.Run()
	case config.Server.FullCommand():
		srv := server.NewServer(
			config.IP.String(),
			*config.Port,
			*config.APIPort,
			*config.PDU,
			*config.ReportInterval,
			*config.KeepaliveInterval,
			*config.InitialMessageID,
			*config.File,
		)
		exitCode = srv.Run()
	case config.Simulator.FullCommand():
		sim := connection.NewSimulator(
			config.IP.String(),
			*config.Port,
			*config.PDU,
			*config.ReportInterval,
			*config.SimulationDir,
			*config.InitialMessageID,
		)
		exitCode = sim.Run()
	}

	os.Exit(exitCode)
}
