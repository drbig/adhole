// See LICENSE.txt for licensing information.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
)

// Source represents a remote data source and holds a function
// that will extract a domain name from a line of data.
type Source struct {
	Name      string
	Desc      string
	URL       string
	Extractor func(string) *string
}

var (
	domains map[string]bool
	wg      sync.WaitGroup
)

// Process will fetch the source data, extract domain names
// and put them into the global domains map.
func (s *Source) Process() {
	fmt.Fprintf(os.Stderr, "Processing list for %s\n", s.Name)
	defer wg.Done()
	resp, err := http.Get(s.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: (%s) %s\n", s.Name, err)
		return
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		domain := s.Extractor(scanner.Text())
		if domain != nil {
			domains[*domain] = true
		}
	}
	return
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] list|all|source source source...\n\n"+
			"list   - show available sources\n"+
			"all    - combine all sources\n"+
			"source - use only specified source(s)\n\n",
			os.Args[0])
		flag.PrintDefaults()
		return
	}
	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	domains = make(map[string]bool, 4096)
	available := make(map[string]*Source, len(sources))
	for _, src := range sources {
		if _, exists := available[src.Name]; exists {
			panic(fmt.Sprintf("Duplicate source %s!", src.Name))
		}
		available[src.Name] = src
	}

	switch flag.Arg(0) {
	case "all":
		for _, src := range sources {
			wg.Add(1)
			go src.Process()
		}
	case "list":
		fmt.Fprintf(os.Stderr, "There are %d sources available:\n", len(sources))
		for i, src := range sources {
			fmt.Fprintf(os.Stderr, "%02d - %-10s - %s\n", i+1, src.Name, src.Desc)
		}
		return
	default:
		for _, name := range flag.Args() {
			src, exists := available[name]
			if !exists {
				fmt.Fprintf(os.Stderr, "Source '%s' not found, skipping...\n", name)
				continue
			}
			wg.Add(1)
			go src.Process()
		}
	}

	wg.Wait()
	fmt.Fprintf(os.Stderr, "Got %d domains total\n", len(domains))
	for domain, _ := range domains {
		fmt.Println(domain)
	}
}
