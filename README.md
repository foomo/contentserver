# Content Server

[![Build Status](https://github.com/foomo/contentserver/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/contentserver/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/contentserver)](https://goreportcard.com/report/github.com/foomo/contentserver)
[![Coverage Status](https://coveralls.io/repos/github/foomo/contentserver/badge.svg?branch=main&)](https://coveralls.io/github/foomo/contentserver?branch=main)
[![GoDoc](https://godoc.org/github.com/foomo/contentserver?status.svg)](https://godoc.org/github.com/foomo/contentserver)

Serves content tree structures very quickly.

## Concept

A Server written in GoLang to mix and resolve content from different content sources, e.g. CMS, Blog, Shop and many
other more. The server provides a simple to use API for non blocking content repository updates, to resolve site content
by an URI or to get deep-linking multilingual URIs for a given contentID.

It's up to you how you use it and which data you want to export to the server. Our intention was to write a fast and
cache hazzle-free content server to mix different content sources.

### Overview

<img src="docs/assets/Overview.svg" width="100%" height="500">

## Export Data

All you have to do is to provide a tree of content nodes as a JSON encoded RepoNode.

| Attribute     |          Type          |                                                 Usage |
|---------------|:----------------------:|------------------------------------------------------:|
| Id            |         string         |                        unique id to identify the node |
| MimeType      |         string         | mime-type of the node, e.g. text/html, image/png, ... |
| LinkId        |         string         |                 (symbolic) link/alias to another node |
| Groups        |        []string        |                                        access control |
| URI           |         string         |                                                   URI |
| Name          |         string         |                                                  name |
| Hidden        |          bool          |                                          hide in menu |
| DestinationId |         string         |                              alias / symlink handling |
| Data          | map[string]interface{} |                                          payload data |
| Nodes         |  map[string]*RepoNode  |                                           child nodes |
| Index         |        []string        |                        contains the order of of nodes |

### Tips

- If you do not want to build a multi-market website define a generic market, e.g. call it *universe*
- keep it lean and do not export content which should not be accessible at all, e.g. you are working on a super secret
  fancy new category of your website
- Hidden nodes can be resolved by their uri, but are hidden on nodes
- To avoid duplicate content provide a DestinationId ( = ContentId of the node you want to reference) instead of URIs

## Request Data

There is a PHP Proxy implementation for foomo in [Foomo.ContentServer](https://github.com/foomo/Foomo.ContentServer).
Feel free to use it or to implement your own proxy in the language you love. The API should be easily to implement in
every other framework and language, too.

## Update Flowchart

<img src="docs/assets/Update-Flow.svg" width="100%" height="700">

### Usage

```bash
$ contentserver -h
Usage of contentserver:
  -address string
    	address to bind socket server host:port
  -debug
    	toggle debug mode
  -free-os-mem int
    	free OS mem every X minutes
  -heap-dump int
    	dump heap every X minutes
  -var-dir string
    	where to put my data (default "/var/lib/contentserver")
  -version
    	version info
  -webserver-address string
    	address to bind web server host:port, when empty no webserver will be spawned
  -webserver-path string
    	path to export the webserver on - useful when behind a proxy (default "/contentserver")
```

## License

Copyright (c) foomo under the LGPL 3.0 license.
