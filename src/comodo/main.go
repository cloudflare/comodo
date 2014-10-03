// comodo: see README for details
//
// Copyright (c) 2014 CloudFlare, Inc.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"os"
	"strings"
	"time"
)

var resolver *string

type work struct {
	domainName string // The domain name to check
    md5 string        // Magic MD5 value
	sha1 string       // Magic SHA1 value
	orderid string    // The Comodo orderid

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

	return fmt.Sprintf("%s,%s,%d,%s", w.domainName, status, w.when.Unix(), w.orderid)
}

func worker(in, out chan work, done chan bool) {
	c := dns.Client{}

	for z := range in {

		// Construct the input name from the MD5 value and the domain name

		z.when = time.Now()
		name := z.md5 + "." + z.domainName + "."

		var cname *dns.Msg
		m := &dns.Msg{}
		m.SetQuestion(name, dns.TypeCNAME)
		cname, _, z.err = c.Exchange(m, *resolver + ":53")

		// If no network error then check that the CNAME points to the SHA1
		// version of the domain name

		if z.err == nil {
			if cname.Rcode == dns.RcodeSuccess && len(cname.Answer) == 1 {
				z.good = strings.HasSuffix(cname.Answer[0].String(),
					"CNAME\t" + z.sha1 + ".comodoca.com.")
			} else {
				z.err = fmt.Errorf("DNS Rcode: %s, Answers: %d",
					dns.RcodeToString[cname.Rcode],
					len(cname.Answer))
			}
		}

		out <- z
	}

	done <- true
}

func filler(in chan work) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		// Expected format is domainname,md5,sha1,orderid

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			fmt.Fprintf(os.Stderr, "Bad input line %s", s.Text())
		} else {
			for i := 0; i < len(parts); i++ {
				parts[i] = strings.TrimSpace(parts[i])
			}
			if len(parts) == 3 {
				parts = append(parts, "na")
			}

			in <- work{domainName: parts[0], md5: parts[1], sha1: parts[2], orderid: parts[3]}
		}
	}

	if err := s.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading standard input: %s\n", err)
	}

	close(in)
}

func main() {
	workers := flag.Int("workers", 10, "Number of workers to run")
	resolver = flag.String("resolver", "8.8.8.8",
		"Address host or host:port of DNS resolver")
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
