// See LICENSE.txt for licensing information.

package main

import (
	"bufio"
	"bytes"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// query wraps Host name and clients UDPAddr.
type query struct {
	Host string
	From *net.UDPAddr
}

// String prints human-readable representation of a query.
func (q *query) String() string {
	return fmt.Sprintf("from %s about %s", q.From, q.Host)
}

// toggle is a synced bool wrapper for expvar.
type toggle struct {
	mu sync.RWMutex
	b  bool
}

// String converts a toggle to string.
func (t *toggle) String() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.b {
		return "true"
	}
	return "false"
}

// Value returns a toggle value.
func (t *toggle) Value() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.b
}

// Toggle toggles a toggle.
func (t *toggle) Toggle() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = !t.b
	return t.b
}

// Flags.
var (
	flagVerbose  = flag.Bool("v", false, "be verbose")
	flagHTTPPort = flag.Int("hport", 80, "HTTP server port")
	flagDNSPort  = flag.Int("dport", 53, "DNS server port")
	flagTimeout  = flag.Duration("t", 5*time.Second, "upstream query timeout")
)

// Expvar exported statistics counters.
var (
	cntMsgs     = expvar.NewInt("statsQuestions")
	cntRelayed  = expvar.NewInt("statsRelayed")
	cntBlocked  = expvar.NewInt("statsBlocked")
	cntTimedout = expvar.NewInt("statsTimedout")
	cntServed   = expvar.NewInt("statsServed")
	cntErrors   = expvar.NewInt("statsErrors")
	cntRules    = expvar.NewInt("statsRules")
)

// 'Static' variables.
// answer will become static once the proxy address bytes have been added.
var (
	// answer is the header of a DNS query response without the domain name and
	// the resource data.
	//
	// Good sources of information on the DNS protocol can be found at:
	// http://www.firewall.cx/networking-topics/protocols/domain-name-system-dns
	// http://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml
	//
	// Bytes described:
	// 2 - Type        = 0x0001     - A
	// 2 - Class       = 0x0001     - IN
	// 4 - TTL         = 0xffffffff - if anyone respects it, this should reduce hits
	// 2 - Data Length = 0x0004     - number of resource bytes, which for IN A IPv4 address is exactly 4
	answer = []byte("\x00\x01\x00\x01\xff\xff\xff\xff\x00\x04")

	// pixel is a hex representation of an 'empty' 1x1 GIF image.
	pixel = "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\xff\xff" +
		"\xff\x00\x00\x00\x21\xf9\x04\x01\x00\x00\x00\x00\x2c\x00\x00" +
		"\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3b"
)

var (
	proxy    *net.UDPConn
	upstream *net.UDPConn
	queries  map[int]*query
	blocked  map[string]bool
	blocking = &toggle{b: true}
	key      string
	list     string
)

func init() {
	expvar.Publish("stateIsRunning", blocking)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] key upstream proxy list.txt\n\n"+
			"key      - password used for /debug actions protection\n"+
			"upstream - real upstream DNS address, e.g. 8.8.8.8\n"+
			"proxy    - servers' bind address, e.g. 127.0.0.1\n"+
			"list.txt - text file with domains to block\n\n",
			os.Args[0],
		)
		flag.PrintDefaults()
		return
	}
	flag.Parse()

	if len(flag.Args()) < 4 {
		flag.Usage()
		os.Exit(1)
	}

	key = flag.Arg(0)
	upIP := parseIPv4(flag.Arg(1), "upstream")
	proxyIP := parseIPv4(flag.Arg(2), "proxy")
	answer = append(answer, proxyIP...)
	list = flag.Arg(3)
	parseList(list)

	var err error
	upAddr := &net.UDPAddr{IP: upIP, Port: 53}
	upstream, err = net.DialUDP("udp4", nil, upAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(2)
	}
	defer upstream.Close()

	proxyAddr := &net.UDPAddr{IP: proxyIP, Port: *flagDNSPort}
	proxy, err = net.ListenUDP("udp4", proxyAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(2)
	}
	defer proxy.Close()

	queries = make(map[int]*query, 4096)

	go runServerHTTP(proxyIP.String())
	go runServerUpstreamDNS()
	go runServerLocalDNS()

	sigwait()
}

// parseIPv4 parses a string to an IPv4 address or dies.
func parseIPv4(arg string, msg string) (ip net.IP) {
	ip = net.ParseIP(arg)
	if ip == nil {
		fmt.Fprintf(os.Stderr, "ERROR: Can't parse %s IP '%s'\n", msg, arg)
		os.Exit(2)
	}
	ip = ip.To4()
	if ip == nil {
		fmt.Fprintln(os.Stderr, "ERROR: IPv6 is not supported, sorry")
		os.Exit(3)
	}
	return
}

// parseList loads a block list file into blocked and updates rules counter.
func parseList(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(2)
	}
	defer file.Close()

	blocked = make(map[string]bool, 4096)
	counter := 0
	scn := bufio.NewScanner(file)
	for scn.Scan() {
		counter++
		blocked[scn.Text()+"."] = true
	}
	log.Printf("DNS: Parsed %d entries from list\n", counter)
	cntRules.Set(int64(counter))
	return
}

// runServerLocalDNS listens for incoming DNS queries and dispatches them for processing.
func runServerLocalDNS() {
	log.Println("DNS: Started local server at", proxy.LocalAddr())

	buf := make([]byte, 512)
	for {
		n, addr, err := proxy.ReadFromUDP(buf)
		if err != nil {
			log.Println("DNS ERROR (1):", err)
			cntErrors.Add(1)
			continue
		}

		msg := make([]byte, n)
		copy(msg, buf[:n])
		cntMsgs.Add(1)
		go handleDNS(msg, addr)
	}
}

// runServerUpstreamDNS listens for upstream answers and relies them to original clients.
func runServerUpstreamDNS() {
	log.Println("DNS: Started upstream server")

	buf := make([]byte, 512)
	for {
		n, _, err := upstream.ReadFromUDP(buf)
		if err != nil {
			log.Println("DNS ERROR (2):", err)
			cntErrors.Add(1)
			continue
		}

		id := int(uint16(buf[0])<<8 + uint16(buf[1]))
		if query, ok := queries[id]; ok {
			delete(queries, id)
			_, err := proxy.WriteTo(buf[:n], query.From)
			if err != nil {
				log.Printf("DNS ERROR: Query id %d %s %s", id, query, err)
				cntErrors.Add(1)
				continue
			}
			if *flagVerbose {
				log.Println("DNS: Relayed answer to query", id)
			}
			cntRelayed.Add(1)
		}
	}
}

// handleDNS peeks the query and either relies it to the upstream DNS server or returns
// a static answer with the 'fake' IP.
func handleDNS(msg []byte, from *net.UDPAddr) {
	var domain bytes.Buffer
	var block bool

	id := int(uint16(msg[0])<<8 + uint16(msg[1]))
	if *flagVerbose {
		log.Printf("DNS: Query id %d from %s\n", id, from)
	}

	count := uint8(msg[5]) // question counter
	offset := 12           // point to first domain name

	if count != 1 {
		log.Printf("DNS WARN: Query id %d from %s has %d questions\n", id, from, count)
		return
	}

	for {
		length := int8(msg[offset])
		if length == 0 {
			break
		}
		offset++
		domain.WriteString(string(msg[offset : offset+int(length)]))
		domain.WriteString(".")
		offset += int(length)
	}
	host := domain.String()
	testHost := host
	parts := strings.Split(testHost, ".")
	try := 1
	for {
		if _, ok := blocked[testHost]; ok {
			block = true
			break
		}
		parts = parts[1:]
		if len(parts) < 3 {
			break
		}
		testHost = strings.Join(parts, ".")
		try++
	}

	if blocking.Value() && block {
		if *flagVerbose {
			log.Printf("DNS: Blocking (%d) %s\n", try, host)
		}
		cntBlocked.Add(1)

		msg[2] = uint8(129) // flags upper byte
		msg[3] = uint8(128) // flags lower byte
		msg[7] = uint8(1)   // answer counter

		msg = append(msg, msg[12:12+1+len(host)]...) // domain
		msg = append(msg, answer...)                 // payload
		_, err := proxy.WriteTo(msg, from)
		if err != nil {
			log.Println("DNS ERROR (3):", err)
			cntErrors.Add(1)
			return
		}
		if *flagVerbose {
			log.Println("DNS: Sent fake answer")
		}
	} else {
		if *flagVerbose {
			log.Println("DNS: Asking upstream")
		}
		queries[id] = &query{From: from, Host: host}
		_, err := upstream.Write(msg)
		if err != nil {
			log.Println("DNS ERROR (4):", err)
			cntErrors.Add(1)
			delete(queries, id)
			return
		}
		go func(queryID int) {
			time.Sleep(*flagTimeout)
			if query, ok := queries[queryID]; ok {
				fmt.Printf("DNS WARN: Query id %d %s timed out\n", queryID, query)
				cntTimedout.Add(1)
				delete(queries, queryID)
			}
			return
		}(id)
	}
	return
}

// authHTTP checks if user supplied proper key.
func authHTTP(req *http.Request) bool {
	if val := req.FormValue("key"); val == key {
		return true
	}
	log.Printf("HTTP: Unauthorized access to %s from %s\n", req.RequestURI, req.RemoteAddr)
	return false
}

// handleHTTP returns an 'empty' 1x1 GIF image for any URL.
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	if *flagVerbose {
		log.Printf("HTTP: Request %s %s %s\n", req.Method, req.Host, req.RequestURI)
	}
	cntServed.Add(1)
	w.Header()["Content-type"] = []string{"image/gif"}
	io.WriteString(w, pixel)
	return
}

// handleReload reloads the rules and redirects to the debug page.
func handleReload(w http.ResponseWriter, req *http.Request) {
	if authHTTP(req) {
		parseList(list)
		log.Println("Rules reloaded:", cntRules)
	}
	http.Redirect(w, req, "/debug/vars", http.StatusSeeOther)
	return
}

// handleToggle toggles blocking and redirects to the debug page.
func handleToggle(w http.ResponseWriter, req *http.Request) {
	if authHTTP(req) {
		log.Println("Blocking toggled to:", blocking.Toggle())
	}
	http.Redirect(w, req, "/debug/vars", http.StatusSeeOther)
	return
}

// runServerHTTP starts the HTTP server.
func runServerHTTP(host string) {
	addr := fmt.Sprintf("%s:%d", host, *flagHTTPPort)
	http.HandleFunc("/", handleHTTP)
	http.HandleFunc("/debug/reload", handleReload)
	http.HandleFunc("/debug/toggle", handleToggle)
	log.Println("HTTP: Started at", addr)
	log.Fatalln(http.ListenAndServe(addr, nil))
	panic("not reachable")
}

// vim: ts=4 sw=4 sts=4
