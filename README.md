Content Server
==============

> Serves content tree structures very quickly through a json socket api

Concept
-------

A Server written in GoLang to mix and resolve content from different content sources, e.g. CMS, Blog, Shop and many other more. The server provides a simple to use API for non blocking content repository updates, to resolve site content by an URI or to get deep-linking multilingual URIs for a given contentID.

It's up to you how you use it and which data you want to export to the server. Our intention was to write a fast and cache hazzle-free content server to mix different content sources.

Export Data
-----------

All you have to do is to provide a tree of content nodes as a JSON encoded RepoNode.

| Attribute     | Type                   | Usage                                                                             |
|---------------|:----------------------:|----------------------------------------------------------------------------------:|
| Id            |         string         |                                                    unique id to identify the node |
| MimeType      |         string         |                             mime-type of the node, e.g. text/html, image/png, ... |
| LinkId        |         string         |                                             (symbolic) link/alias to another node |
| Groups        |        []string        |                                                                    access control |
| URI           |         string         | a map of unique URIs for each region and language to resolve and link to the node |
| Name          |         string         |                                 a name for this node in every region and language |
| Hidden        |          bool          |                                     hide in menu specific for region and language |
| DestinationId |         string         |                                                         alias or symlink handling |
| Data          | map[string]interface{} |                                                                      payload data |
| Nodes         |  map[string]*RepoNode  |                                                                       child nodes |
| Index         |        []string        |                                                    contains the order of ou nodes |

### Tips

-	If you do not want to build a multi-market website define a generic market, e.g. call it *universe*.
-	Do not export content which should not be accessible at all, e.g. you are working on a super secret fancy new category of your website.
-	Hidden nodes could be resolved by uri-requests, but are not served on a navigation node request.
-	To make a node accessible only for a given region/language is totally easy, just set the region/language you want to serve. Other regions and languages will not contain this node any more.
-	To avoid duplicate content provide a DestinationId ( = ContentId of the node you want to reference) instead of URIs.

Request Data
------------

There is a PHP Proxy implementation for foomo in [Foomo.ContentServer](https://github.com/foomo/Foomo.ContentServer). Feel free to use it or to implement your own proxy in the language you love. The API should be easily to implement in every other framework and language, too.

Usage
-----

```
$ contentserver --help

Usage of bin/contentserver:
  -address="127.0.0.1:8081": address to bind host:port
  -logLevel="record": one of error, record, warning, notice, debug
  -protocol="tcp": what protocol to server for
  -vardir="127.0.0.1:8081": where to put my data
```

Packaging & Deployment
----------------------

In order to build packages and upload to Package Cloud, please install the following requirements and run the make task.

[Package Cloud Command Line Client](https://packagecloud.io/docs#cli_install)

```
$ gem install package_cloud
```

[FPM](https://github.com/jordansissel/fpm)

```
$ gem install fpm
```

Building package

```
$ make package
```

*NOTE: you will be prompted for Package Cloud credentials.*

Testing
-------

```
$ git clone https://github.com/foomo/contentserver.git
$ cd contentserver
$ make test
```

Contributing
------------

In lieu of a formal styleguide, take care to maintain the existing coding style. Add unit tests and examples for any new or changed functionality.

1.	Fork it
2.	Create your feature branch (`git checkout -b my-new-feature`\)
3.	Commit your changes (`git commit -am 'Add some feature'`\)
4.	Push to the branch (`git push origin my-new-feature`\)
5.	Create new Pull Request

License
-------

Copyright (c) foomo under the LGPL 3.0 license.
