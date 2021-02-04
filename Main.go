/*-----------------------------------------------------------------------------------
  --  RETTER                                                                       --
  --  Copyright (C) 2021  RETTER's Contributors                                    --
  --                                                                               --
  --  This program is free software: you can redistribute it and/or modify         --
  --  it under the terms of the GNU Affero General Public License as published     --
  --  by the Free Software Foundation, either version 3 of the License, or         --
  --  (at your option) any later version.                                          --
  --                                                                               --
  --  This program is distributed in the hope that it will be useful,              --
  --  but WITHOUT ANY WARRANTY; without even the implied warranty of               --
  --  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                --
  --  GNU Affero General Public License for more details.                          --
  --                                                                               --
  --  You should have received a copy of the GNU Affero General Public License     --
  --  along with this program.  If not, see <https:   -- www.gnu.org/licenses/>.   --
  -----------------------------------------------------------------------------------*/

package main

import (
	"context"
	"fmt"
	"github.com/hyperjumptech/jiffy"
	"github.com/hyperjumptech/retter/test"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func splash() {
	fmt.Println("________________________________________________________________ ")
	fmt.Println(`\______   \_   _____/\__    ___/\__    ___/\_   _____/\______   \`)
	fmt.Println(" |       _/|    __)_   |    |     |    |    |    __)_  |       _/")
	fmt.Println(` |    |   \|        \  |    |     |    |    |        \ |    |   \`)
	fmt.Println(" |____|_  /_______  /  |____|     |____|   /_______  / |____|_  /")
	fmt.Println(`        \/        \/                               \/         \/ `)
}

func main() {
	if len(os.Args) == 1 {
		splash()
		startServer()
	} else if len(os.Args) == 2 {
		test.StartDummyServer(os.Args[1], true)
	}
}

func startServer() {
	startTime := time.Now()
	listen := Config.GetString(ServerListen)
	if len(listen) == 0 {
		panic("server.listen not configured")
	}

	if len(listen) == 0 {
		panic("backend.baseurl not configured")
	}

	var wait time.Duration

	graceShut, err := jiffy.DurationOf(Config.GetString("server.timeout.graceshut"))
	if err != nil {
		panic(err)
	}
	wait = graceShut
	WriteTimeout, err := jiffy.DurationOf(Config.GetString("server.timeout.write"))
	if err != nil {
		panic(err)
	}
	ReadTimeout, err := jiffy.DurationOf(Config.GetString("server.timeout.read"))
	if err != nil {
		panic(err)
	}
	IdleTimeout, err := jiffy.DurationOf(Config.GetString("server.timeout.idle"))
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr: listen,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: WriteTimeout,
		ReadTimeout:  ReadTimeout,
		IdleTimeout:  IdleTimeout,
		Handler:      NewRetterHTTPHandler(), // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		fmt.Printf("Retter is listening on : %s\n", listen)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	dur := time.Now().Sub(startTime)
	durDesc := jiffy.DescribeDuration(dur, jiffy.NewWant())
	log.Infof("Shutting down. This RETTER been protecting the backend service for %s", durDesc)
	os.Exit(0)
}
