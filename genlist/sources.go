// See LICENSE.txt for licensing information.

package main

import (
	"regexp"
)

var (
	easylistRegexp *regexp.Regexp
)

// put any extractor global stuff here.
func init() {
	easylistRegexp = regexp.MustCompile("^\\|\\|((\\w+\\.)+[[:alpha:]]+)[^[:alpha:]]")
}

// sources defines an array of available blacklist sources.
var sources = [...]*Source{
	&Source{
		Name: "pgl",
		Desc: "http://pgl.yoyo.org/adservers",
		URL:  "http://pgl.yoyo.org/adservers/serverlist.php?hostformat=nohtml&showintro=%0A0&startdate%5Bday%5D=&startdate%5Bmonth%5D=&startdate%5Byear%5D=&mimetype=plaint%0Aext",
		Extractor: func(line string) *string {
			return &line
		},
	},
	&Source{
		Name: "easylist",
		Desc: "https://easylist.adblockplus.org",
		URL:  "https://easylist-downloads.adblockplus.org/easylist.txt",
		Extractor: func(line string) *string {
			match := easylistRegexp.FindStringSubmatch(line)
			if match == nil {
				return nil
			}
			return &match[1]
		},
	},
}
