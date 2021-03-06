// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/discover"
	"github.com/syncthing/syncthing/lib/protocol"
)

var timeout = 5 * time.Second

func main() {
	var server string

	flag.StringVar(&server, "server", "", "Announce server (blank for default set)")
	flag.DurationVar(&timeout, "timeout", timeout, "Query timeout")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(64)
	}

	id, err := protocol.DeviceIDFromString(flag.Args()[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if server != "" {
		checkServers(id, server)
	} else {
		checkServers(id, config.DefaultDiscoveryServers...)
	}
}

type checkResult struct {
	server string
	direct []string
	relays []discover.Relay
	error
}

func checkServers(deviceID protocol.DeviceID, servers ...string) {
	t0 := time.Now()
	resc := make(chan checkResult)
	for _, srv := range servers {
		srv := srv
		go func() {
			res := checkServer(deviceID, srv)
			res.server = srv
			resc <- res
		}()
	}

	for _ = range servers {
		res := <-resc

		u, _ := url.Parse(res.server)
		fmt.Printf("%s (%v):\n", u.Host, time.Since(t0))

		if res.error != nil {
			fmt.Println("  " + res.error.Error())
		}
		for _, addr := range res.direct {
			fmt.Println("  address:", addr)
		}
		for _, rel := range res.relays {
			fmt.Printf("  relay: %s (%d ms)\n", rel.URL, rel.Latency)
		}
	}
}

func checkServer(deviceID protocol.DeviceID, server string) checkResult {
	disco, err := discover.NewGlobal(server, tls.Certificate{}, nil, nil)
	if err != nil {
		return checkResult{error: err}
	}

	res := make(chan checkResult, 1)

	time.AfterFunc(timeout, func() {
		res <- checkResult{error: errors.New("timeout")}
	})

	go func() {
		direct, relays, err := disco.Lookup(deviceID)
		res <- checkResult{direct: direct, relays: relays, error: err}
	}()

	return <-res
}

func usage() {
	fmt.Printf("Usage:\n\t%s [options] <device ID>\n\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
}
