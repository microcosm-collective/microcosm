# Microcosm

Microcosm is a community CMS designed as a platform to power multiple bulletin-boards, forums and community sites.

This repo is part of a collection of repos that comprise the Microcosm product. This specific repo covers the core API and acts as middleware between the storage (PostgreSQL and Amazon S3) and the web application (Django).

The API handles all permissions, caching and is designed to be world-readable with developers encouraged to build against the API.

The API is written in Go.

## Layout

Layout was loosely inspired by Django and Rails with the aim of providing a vaguely familiar environment to developers new to the project.

* `audit` = Internal API to records IP addresses for data-changing tasks, used for spam management
* `cache` = Internal API to place items into memcached
* `config` = Internal API to load and parse the `/etc/microcosm/api.conf` config file
* `controller` = Public RESTful API to provide the full functionality of Microcosm in a consistent and easy to use JSON interface
* `errors` = Internal API to create custom errors wherever the HTTP error codes are insufficient to describe the full nature of an error
* `helpers` = Utility library that crept in and provides some common functions and types to the wider code base
* `models` = Internal CRUD API and query API for saving data to the storage layer(s) and to query and retrieve that data
* `redirector` = Public API for handling short URLs and redirecting (with optional re-write) URLs, used for affiliate revenue and malware blocking (block bad domains)
* `resolver` = Public API that handles 404 not found URLs for a client, determines whether the URL matches a known pattern for the site, and if so will provide a new URL that locates the data. Used after importing a site, to ensure old URLs continue to work.
* `server` = Registers web handlers and cron jobs

## Dependencies

For the database migrations you will need [goose](https://github.com/pressly/goose) installed:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Config File

For the microcosm binary to run, /etc/microcosm/api.conf needs to exist with the following configuration keys (values obviously changed for your environment):

```
[api]

listen_port=8080
microcosm_domain=microcosm.app

database_host=sql.dev.microcosm.cc
database_port=5432
database_database=microcosm
database_username=microcosm
database_password=thePassword

memcached_host=127.0.0.1
memcached_port=11211
```

*listen_port* is the port that the microcosm daemon will listen and serve HTTP requests to the world (or probably to the load balancer).

*microcosm_domain* is the domain that is sub-domained. So if meta.microcosm.app is the site, then microcosm.app is the microcosm_domain.

## Design Principles

The vast majority of the design is in the [documentation](http://microcosm-cc.github.io/), however there are some principles that are not surfaced through the front-end and they are captured here and need to be considered when authoring new API endpoints or modifying existing ones.

### Draft the calls before writing code

APIs serve the developer, put yourself in the developer shoes and think long and hard about what the API seeks to accomplish, why you want to call it and what you are trying to achieve.

Think about:

1. What the developer is trying to achieve
1. How you can minimise the number of calls the developer needs to make
1. How you can make the API intuitive (consistent and predictable)
1. How you can help reduce the amount of code the developer needs to write (re-usable structures from other resource endpoints)
1. How a web client will call the API
1. How a mobile client will call the API
1. How a batch processing client will call the API

Before you write any real code you should produce a single text file demonstrating some ideal cURL requests and the responses from those, listing the querystring arguments that can be used.

You should get this document approved by your peers.

It is much cheaper and quicker to change things when they are just an idea than it is to change things once you've written code. It is extraordinarily expensive to change things once they have been made available and *any* client has coded to the API. Get it right before you write code.

### Everything is a resource

There are no trailing slashes on URLs as resources are not directories.

When thinking of operations that are verbs, do not create verb operation URLs, instead look at how we use PATCH or consider a child resource. Examples: To open a thread is performed by PATCH, and to attend an event is performed by POSTing to an attendees resource that is a child of the event.

### Cache is atomic and internal

Caching is easy, but cache invalidation is hard.

For this reason we are following a principle of only ever caching something that is identifiable by a single key (e.g. the Id of the item) and that does not change from user to user.

Guidelines: Cache every item, never cache a collection. Collections should be created by querying keys, and then fetching the individual items from cache.

Almost every request to the API returns information on the user's permissions. As permissions *could* change and it is important to reflect permissions changes quickly, we need to ensure that caching is *behind* the authentication/authorisation layer.

What this effectively means is that nothing is cached in HTTP external to the microcosm application, and any caches that exist must be behind the authorisation layer and within our application.

Caches make individual item requests cheap.

## Patterns in HTTP methods

### OPTIONS

Should exist for every resource, describes available methods

### POST

1. Do a duplicate check before creating
1. Do a spam check at time of creation
1. Flush any parent caches when item is created

### GET

1. Auth check as soon as possible
1. Check visibility of parent items (if applicable) before returning
1. Always get from cache if possible, or fill the cache
1. If fetching a collection, only fetch Id columns and fetch individual items from cache

### UPDATE

1. Auth check as soon as possible
1. Always fetch the thing to be updated prior to updating
1. Flush any parent caches when item is created

### PATCH

1. Auth check as soon as possible
1. Always fetch the thing to be updated
1. Check permission of every patch path (i.e. only moderators may be allowed to sticky something)
1. Flush any parent caches when item is created

### DELETE

1. Auth check as soon as possible
1. Fetch the thing to be deleted (it should come from cache and will save a delete SQL query if it doesn't exist)
1. Always return 200 OK unless 401 or 500

## Performance targets

GET Requests that result in fully cached items and requiring only permissions and context checks should return in less than 10ms.

Requests that cannot be cached and result in very complex operations (PATCH a collection of things, or POST where it performs a chain of INSERTs within a transaction) should return in less than 250ms. That is a worst case for something like "Create Comment", which could be a complex operation (check permissions of parent item, and parent microcosm, parse markdown, render HTML, clean HTML, parse embeds, clean HTML).
