# adhole [![Build Status](https://travis-ci.org/drbig/adhole.svg?branch=master)](https://travis-ci.org/drbig/adhole)

AdHole is a simple transparent advertisement and tracking blocker intended for 
personal use.

It works by providing two elements: first a DNS server that, based on a list of 
domains, either relies the query to a real server, or returns its own IP 
address; and a micro HTTP server that serves an empty GIF image for any 
request. As such its functionality is akin to 
[AdBlock](https://chrome.google.com/webstore/detail/adblock/gighmmpiobklfepjocnamgkkbiglidom?hl=en)
, but less pretty (you'll probably see empty space where ads 
used to be) and completely transparent (i.e. will work for any browser, on any 
OS on any machine, locally or over any network). Note that you can just as well 
block *any* domains.

I use AdHole in 'production' on my 
[RaspberryPi](http://www.raspberrypi.org/)-powered WiFi router, where it 
provides ad and tracking blocking to all my wireless toys.

My programming goal was to make it as simple and minimal as possible. Therefore 
instead of a full DNS stack (and especially parsing) I work directly on 
`[]byte` slices and try to reuse the data already on hand. It was also a nice 
little adventure in dealing with binary protocols the low-level way.

## Changelog

- **2016-11-11** Added locking around the queries map. Can't really test it as the RPi is obviously single-core.

## Building

On a Unix-like system you should be able to get everything built using just:

    $ make
    cd adhole; \
    gofmt -w *.go; \
    go build .
    cd genlist; \
    gofmt -w *.go; \
    go build .

Otherwise just run `go build .` in either or both of `adhole/` and `genlist/`.

## Usage

    $ ./adhole
    Usage: ./adhole [options] key upstream proxy list.txt
    
    key      - password used for /debug actions protection
    upstream - real upstream DNS address, e.g. 8.8.8.8
    proxy    - servers' bind address, e.g. 127.0.0.1
    list.txt - text file with domains to block
    
      -dport=53: DNS server port
      -hport=80: HTTP server port
      -t=5s: upstream query timeout
      -v=false: be verbose

Note that you will need root privileges to run it on the default ports.

List format is simply: one domain name per line. All subdomains of a given 
domain will be blocked, so there is no need to use `*`. Domains should also not 
end with a dot. The parser should also be indifferent to line endings. Example 
list file:

    101com.com
    101order.com
    103bees.com
    123found.com
    123pagerank.com

To get a decent list of domains to block I recommend going 
[here](http://pgl.yoyo.org/adservers/) and generating a 'plain non-HTML list -- 
as a plain list of hostnames (no HTML)' with 'no links back to this page' and 
'view list as plain text:' ticked (that was so verbose...). Or alternatively 
[this](http://pgl.yoyo.org/adservers/serverlist.php?hostformat=nohtml&showintro=0&startdate%5Bday%5D=&startdate%5Bmonth%5D=&startdate%5Byear%5D=&mimetype=plaintext)
 should work as a direct link to the current list...

Or you can use the experimental `genlist` utility. Currently I've implemented 
fetching for the above list and for 
[EasyList](https://easylist.adblockplus.org/en/). 
If you have a better / your favourite source of domain blacklist 
please look at `sources.go` and implement a fetcher. As a side note, sources 
that are meant to be used by AdBlock (or equivalent) are not the best 
candidates here, due to the fact that AdBlock is much more subtle, i.e. with 
current extraction method a list generated with EasyList input will block 
google.com, bbc.co.uk and imdb.com (and probably many more sites that you 
care about). Also the whole thing is brittle by nature, hence the permanent 
'experimental' status.

    $ ./genlist
    Usage: ./genlist [options] list|all|source source source...
    
    list   - show available sources
    all    - combine all sources
    source - use only specified source(s)
    
    $ ./genlist list
    There are 2 sources available:
    01 - pgl        - http://pgl.yoyo.org/adservers
    02 - easylist   - https://easylist.adblockplus.org
    
    $ ./genlist all > /tmp/blacklist.txt
    Processing list for pgl
    Processing list for easylist
    Got 9829 domains total

Example [systemd](http://www.freedesktop.org/wiki/Software/systemd/) service 
file:

    [Unit]
    Description=DNS blackhole
    After=network.target
    
    [Service]
    ExecStart=/home/user/adhole SecretKey 192.168.0.3 192.168.0.21 /home/user/blacklist.txt
    
    [Install]
    WantedBy=multi-user.target

Thanks to the great [expvar](http://golang.org/pkg/expvar/) package you can 
monitor some statistics by visiting `http://proxy.addr/debug/vars`. The 
following items are relevant:

  * `stateIsRunning` - if false all queries are relied to upstream
  * `statsQuestions` - number of received queries
  * `statsRelayed` - number of queries relayed to the real server
  * `statsBlocked` - number of queries blocked
  * `statsTimedout` - number of relayed queries that timed out
  * `statsServed` - number of HTTP requests served
  * `statsErrors` - number of errors encountered
  * `statsRules` - number of items read from the blacklist

You can also do the following actions via HTTP:

  * `/debug/reload` - will reload the list.txt file
  * `/debug/toggle` - toggle blocking on and off

You'll need to append `&key=YOURKEY` to the above. Unauthorized hits will 
be logged. Note that you may set the key to `""` (i.e. an empty key) and 
therefore disable the authentication.

**Tested on:**

  * Linux - amd64, armv6l
  * Windows XP - i686

## Contributing

Feel free to either use GitHub's pull request or send me patches directly.

I also welcome any feedback regarding stability and what OS/platform you run 
AdHole on (just out of curiosity).

## Bugs / Todo

  * Edge cases (multiple questions per query, anybody?)
  * Even less data shuffling
  * IPv4 only

## Copyright

Copyright (c) 2014 - 2016 Piotr S. Staszewski

Absolutely no warranty. See LICENSE.txt for details.
