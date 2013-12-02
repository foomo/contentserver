Content2Go Server
===========

## Concept

A Server written in GoLang to mix and resolve content from different content sources, e.g. CMS, Blog, Shop and many other more. The server provides a simple to use API for non blocking content repository updates, to resolve site content by an URI or to get deep-linking multilingual URIs for a given contentID.

It's up to you how you use it and which data you want to export to the server. Our intention was to write a fast and cache hazzle-free content server to mix different content sources.

## Export Data

All you have to do is to provide a tree of content nodes as a JSON encoded RepoNode.

| Attribute | Type | Usage |
| ------------- |:-------------:| -----:|
| Id | string | unique id to identify the node |
| Updated | int | last modification date as unix timestamp |
| MimeType | string | mime-type of the node, e.g. text/html, image/png, ... |
| Handler | string | define a handler to easily control the output rendering |
| Regions | []string | define regions for multi-market websites |
| URIs | map[string]map[string]string | a map of unique URIs for each region and language to resolve and link to the node |
| DestinationId | string | alias or symlink handling |
| Names | map[string]map[string]string | a name for this node in every region and language |
| Hidden | map[string]map[string]bool | hide in menu specific for region and language |
| Groups | []string | access control |
| Data | map[string]interface{} | payload data |
| Content | map[string]interface{} | ??? |
| Nodes | map[string]*RepoNode | child nodes |
| Index | []string | ??? |
| LinkIds | map[string]map[string]string | ??? |

### Tips

* If you do not want to build a multi-market website define a generic market, e.g. call it _universe_.
* Do not export content which should not be accessible at all, e.g. you are working on a super secret fancy new category of your website.
* Hidden nodes could be resolved by uri-requests, but are not served on a navigation node request.
* To make a node accessible only for a given region/language is totally easy, just set the region/language you want to serve. Other regions and languages will not contain this node any more.
* To avoid duplicate content provide a DestinationId ( = ContentId of the node you want to reference) instead of URIs.

## Request Data

There is a PHP Proxy for our awesome open-source PHP Web-Framework called Foomo. Feel free to use it or to implement your own proxy in the language you love. The API should be easily to implement in every other framework and language, too.