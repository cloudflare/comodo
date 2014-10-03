// comodo: see README for details
//
// Copyright (c) 2014 CloudFlare, Inc.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type work struct {
	domainName string // The domain name to check
    md5 string        // Magic MD5 value
	sha1 string       // Magic SHA1 value

	// Set when the work job has been processed

	good bool         // True if validation worked
	err error         // Non-nil if network error
	when time.Time    // When the lookup was performed
}

// String interface to work so that printf can be used in main() below to
// print out the status of a work job.
func (w work) String() string {
	var status string
	if w.err == nil {
		if w.good {
			status = "OK"
		} else {
			status = "BAD"
		}
	} else {
		status = "ERROR"
	}

	return fmt.Sprintf("%s,%s,%s", w.domainName, status, w.when.UTC())
}

func worker(in, out chan work, done chan bool) {
	for z := range in {

		// Construct the input name from the MD5 value and the domain name

		z.when = time.Now()
		name := z.md5 + "." + z.domainName

		fmt.Printf("%#v\n", z)
		fmt.Printf("%#v\n", name)

		var cname string
		cname, z.err = net.LookupCNAME(name)

		fmt.Printf("%#v\n", cname)
		fmt.Printf("%#v\n", z.err)

		// If no network error then check that the CNAME points to the SHA1
		// version of the domain name

		if z.err == nil {
			z.good = cname == z.sha1 + ".comodoca.com"
		}

		out <- z
	}

	done <- true
}

func filler(in chan work) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {

		// Expected format is domainname,md5,sha1

		parts := strings.Split(strings.TrimSpace(s.Text()), ",")
		if len(parts) != 3 {
			fmt.Fprintf(os.Stderr, "Bad input line %s", s.Text())
		} else {
			for i := 0; i < len(parts); i++ {
				parts[i] = strings.TrimSpace(parts[i])
			}

			in <- work{domainName: parts[0], md5: parts[1], sha1: parts[2]}
		}
	}

	if err := s.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading standard input: %s\n", err)
	}

	close(in)
}

func main() {
	workers := flag.Int("workers", 10, "Number of workers to run")
	flag.Parse()
	
	in := make(chan work)
	out := make(chan work)
	done := make(chan bool)

	for i := 0; i < *workers; i++ {
		go worker(in, out, done)
	}

	go filler(in)

	alive := *workers

	for alive > 0 {
		select {
		case z := <-out:
			fmt.Printf("%s\n", z)

		case <-done:
			alive -= 1 
		}
	}

	close(out)
}
