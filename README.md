[![status-badge](https://ci.codeberg.org/api/badges/15653/status.svg)](https://ci.codeberg.org/repos/15653)
[![License](https://img.shields.io/badge/license-GPL-blue.svg)](https://codeberg.org/scip/ephemerup/raw/branch/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/codeberg.org/scip/ephemerup)](https://goreportcard.com/report/codeberg.org/scip/ephemerup)

# ephemerup
Simple standalone file upload server with expiration and commandline client.

## Introduction

**ephemerup** is a simple standalone  file server where every uploaded
file expires  sooner or later. The  server provides a RESTful  API and
can be used easily with the commandline client `upctl`.

The idea is to provide a way to quickly exchange files between parties
when  no other  way  is available  and the  files  themselfes are  not
important enough to  keep them around. Think of  this szenario: you're
working for  the network departement  and there's a problem  with your
routing. Tech support  asks you to create a network  trace and send it
to  them. But  you  can't because  the  trace file  is  too large  and
sensitive to  be sent by email.  This is where **ephemerup**  comes to
the rescue.

You upload the  file, send the download  url to the other  party and -
assuming you've utilized  the defaults - when they download  it, it is
being deleted  immediately from the  server. But  you can also  set an
expire time, say 5 days or something like that.

The  download urls  generated  by **ephemerup**  consist  of a  unique
onetime  hash, so  they  are somewhat  confident.  However, if  you're
uploading really sensitive data, you better encrypt it.

**ephemerup** also  supports something we  call an API  Context. There
can be many such API contexts.  Each of these has an associated token,
which  has  to be  used  by  legitimate  clients to  authenticate  and
authorize. A user  can only manage uploads within  that context. Think
"tenant" if you will.

## Demo

![demo upctl session](demo/upctl.gif)

## Features

- RESTful API
- Authentication and Authorization through bearer api token
- multiple tenants supported (tenant == api context)
- Each upload gets its own unique id
- download uri is public, no api required, it is intended for end users
- uploads may consist of one or multiple files
- zipped automatically
- uploads expire, either as soon as it gets downloaded or when a timer runs out
- the command line client uses the api
- configuration using HCL language
- docker container build available
- the server supports config by config file, environment variables or flags
- restrictive defaults

## Installation

### Deploy server using pre-built docker file

A   ready   to   use   ephemerup  server   image   is   available   on
[ghcr.io](https://ghcr.io/tlinden/ephemerup).   Supported  tags   are:
`latest` or a github release tag.

To try it locally with docker:

```
docker run -dp 8080:8080 --name eph \
     ghcr.io/tlinden/ephemerup:latest \
    -LogLevel=info
```

### Build Dockerfile

There's a `Dockerfile` available for the server so you can build and run it using docker:
```
make buildimage
docker-compose run ephemerup
```
Then use the client to test it.

### Install from binary package

Go  to  the  [Releases](https://codeberg.org/scip/ephemerup/releases)
page and download the latest tarball  for your platform. Unpack it and
execute `make install` inside the created directory.

This installs both the server `ephemerupd` and the client `upctl`.

If you  only need the client,  just grab the tarball  and extract just
the client, copy it to your bin folder and you're good to go.

### Deploy on Kubernetes using the Helm chart

```
helm repo add tlinden https://tlinden.github.io/ephemerup/
helm repo update
helm upgrade --install ephemerup tlinden/ephemerup --namespace ephemerup --create-namespace
```

Refer to the [chart documentation](https://codeberg.org/scip/ephemerup/tree/main/charts/ephemerup) for help.

For starters, create a minimal `values.yaml` like this one:
```yaml
image:
  tag: "v0.0.3"
```

**Please note that the helm chart doesn't deploy a loadbalancer, you need to do this yourself, if needed.**

### Build from source

To build from source, you'll  need  a go  build  environment.

Clone the git repo
Just run `make` to build everything.


## Server Usage

```
ephemerupd -h
      --apikeys strings     Api key[s] to allow access
  -a, --apiprefix string    API endpoint path (default "/api")
  -n, --appname string      App name to say hi as (default "ephemerupd v0.0.1")
  -b, --bodylimit int       Max allowed upload size in bytes (default 10250000000)
  -c, --config string       custom config file
  -D, --dbfile string       Bold database file to use (default "/tmp/uploads.db")
  -d, --debug               Enable debugging
      --frontpage string    Content or filename to be displayed on / in case someone visits (default "welcome to upload api, use /api enpoint!")
  -4, --ipv4                Only listen on ipv4
  -6, --ipv6                Only listen on ipv6
  -l, --listen string       listen to custom ip:port (use [ip]:port for ipv6) (default ":8080")
  -p, --prefork             Prefork server threads
  -s, --storagedir string   storage directory for uploaded files (default "/tmp")
      --super string        The API Context which has permissions on all contexts
  -u, --url string          HTTP endpoint w/o path
  -v, --version             Print program version
```

All flags can be set using environment variables, prefix the flag with `EPHEMERUPD_` and uppercase it, eg: 
```
EPHEMERUPD_LISTEN=:8080
```

In addition it is possible to set api contexts using env vars (otherwise only possible using the config file):
```
EPHEMERUPD_CONTEXT_SUPPORT="support:tymag-fycyh-gymof-dysuf-doseb-puxyx"
EPHEMERUPD_CONTEXT_FOOBAR="foobar:U3VuIE1hciAxOSAxMjoyNTo1NyBQTSBDRVQgMjAyMwo"
```

Configuration can also be done using a config file (searched in the following locations):
- `/etc/ephemerupd.hcl`
- `/usr/local/etc/ephemerupd.hcl`
- `~/.config/ephemerupd/ephemerupd.hcl`
- `~/.ephemerupd`
- `$(pwd)/ephemerupd.hcl`

Or using the flag `-c`. Sample config file:
```
listen = ":8080"
bodylimit = 10000

apicontext = [
  {
    context = "root"
    key = "0fddbff5d8010f81cd28a7d77f3e38981b13d6164c2fd6e1c3f60a4287630c37",
  },
  {
    context = "foo",
    key = "970b391f22f515d96b3e9b86a2c62c627968828e47b356994d2e583188b4190a"
  }
]

#url = "https://sokrates.daemon.de"

# this is the root context with all permissions
super = "root"
```

### Server endpoint

The   server   serves   the   API  under   the   following   endpoint:
`http://SERVERNAME[:PORT]/api/v1`  where   SERVERNAME[:PORT]  is  the
argument  to  the  `-l`  commandline argument  or  the  config  option
`listen` or the environment variable `EPHEMERUPD_LISTEN`.

By default  the server listens  on any interface  ip4 and ipv6  on TCP
port  8080. You  can  specify a  server  name or  an  ipaddress and  a
port. The server can be configured to run on ipv6 (or ipv4) only using
the `-4` respective the `-6` commandline flags.

It does not  support TLS at the  moment. Use a nginx  reverse proxy in
front of it.

### Server REST API

Every endpoint returns a JSON object. Each returned object contains the data requested plus:

- success: true or false
- code: HTTP Response Code
- message: error message, if success==false

#### Endpoints

| HTTP Method | Endpoint              | Parameters          | Input                      | Returns                               | Description                                   |
|-------------|-----------------------|---------------------|----------------------------|---------------------------------------|-----------------------------------------------|
| GET         | /v1/uploads           | apicontext,q,expire |                            | List of upload objects                | list upload objects                           |
| POST        | /v1/uploads           |                     | multipart-formdata file[s] | List of 1 upload object if successful | upload a file and create a new upload object  |
| GET         | /v1/uploads/{id}      |                     |                            | List of 1 upload object if successful | list one specific upload object matching {id} |
| DELETE      | /v1/uploads/{id}      |                     |                            | Noting                                | delete an upload object identified by {id}    |
| PUT         | /v1/uploads/{id}      |                     | JSON upload object         | List of 1 upload object if successful | modify an upload object identified by {id}    |
| GET         | /v1/uploads/{id}/file |                     |                            | File download                         | Download the file associated with the  object |
| GET         | /v1/forms             | apicontext,q,expire |                            | List of form objects                  | list form objects                             |
| POST        | /v1/forms             |                     | JSON form object           | List of 1 form object if successful   | create a new form object                      |
| GET         | /v1/forms/{id}        |                     |                            | List of 1 form object if successful   | list one specific form object matching {id}   |
| DELETE      | /v1/forms/{id}        |                     |                            | Noting                                | delete an form object identified by {id}      |
| PUT         | /v1/forms/{id}        |                     | JSON form object           | List of 1 form object if successful   | modify an form object identified by {id}      |

#### Consumer URLs

The following endpoints are no API urls, but accessed directly by consumers using their browser or `wget` etc:

| URL                     | Description                                             |
|-------------------------|---------------------------------------------------------|
| /                       | Display a short welcome message, can be customized      |
| /download/{id}[/{file}] | Download link returned after an upload has been created |
| /form/{id}              | Upload form for consumer                                |

#### API Objects

Response:

| Field   | Data Type | Description                           |
|---------|-----------|---------------------------------------|
| success | bool      | if true the request was successful    |
| code    | int       | HTTP response code                    |
| message | string    | error message, if any                 |
| uploads | array     | list of upload objects (may be empty) |
| forms   | array     | list of form objects (may be empty)   |

Upload:

| Field    | Data Type        | Description                                                                                                                                 |
|----------|------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| id       | string           | unique identifier for the object                                                                                                            |
| expire   | string           | when the upload has to expire, either "asap" or a Duration using numbers and the letters d,h,m,s (days,hours,minutes,seconds), e.g. 2d4h30m |
| file     | string           | filename after uploading, this is what a consumer gets when downloading it                                                                  |
| members  | array of strings | list of the original filenames                                                                                                              |
| created  | timestamp        | time of object creation                                                                                                                     |
| context  | string           | the API context the upload has been created under                                                                                           |
| url      | string           | the download URL                                                                                                                            |

Form:

| Field       | Data Type | Description                                                                                                                               |
|-------------|-----------|-------------------------------------------------------------------------------------------------------------------------------------------|
| id          | string    | unique identifier for the object                                                                                                          |
| expire      | string    | when the form has to expire, either "asap" or a Duration using numbers and the letters d,h,m,s (days,hours,minutes,seconds), e.g. 2d4h30m |
| description | string    | arbitrary description, shown on the form page                                                                                             |
| context     | string    | the API context the form has been created under and the uploaded files will be created on                                                 |
| notify      | string    | email address of the form creator, who gets an email once the consumer has uploaded files using the form                                  |
| created     | timestamp | time of object creation                                                                                                                   |
| url         | string    | the form URL                                                                                                                              |

Note: if the expire field for a form  is not set or set to "asap" only
1 upload  object can be created  from it.  However, if  a duration has
been specified, the  form can be used multiple times  and thus creates
multiple upload objects.



## Client Usage

```
upctl 
Error: No command specified!
Usage:
  upctl [options] [flags]
  upctl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  delete      Delete an upload
  describe    Describe an upload.
  download    Download a file.
  form        Form commands
  help        Help about any command
  list        List uploads
  upload      Upload files

Flags:
  -a, --apikey string     Api key to use
  -c, --config string     custom config file
  -d, --debug             Enable debugging
  -p, --endpoint string   upload api endpoint url (default "http://localhost:8080/api/v1")
  -h, --help              help for upctl
  -r, --retries int       How often shall we retry to access our endpoint (default 3)
  -v, --version           Print program version

Use "upctl [command] --help" for more information about a command.
```

The client must be configured using a config file. The following locations are searched for it:
- `$(pwd)/upctl.hcl`
- `~/.config/upctl/upctl.hcl`

Sample config file for a client:
```
endpoint = "http://localhost:8080/api/v1"
apikey = "970b391f22f515d96b3e9b86a2c62c627968828e47b356994d2e583188b4190a"
```

The `endpoint` is  the **ephemerup** server running  somewhere and the
`apikey` is the token you got from the server operator..


## TODO

- add metrics (as in https://github.com/ansrivas/fiberprometheus)
- do not manually generate output urls, use fiber.GetRoute()
- upd: https://docs.gofiber.io/guide/error-handling/ to always use json output
- add (default by time!) sorting to list outputs, and add sort flag


## BUGS

### upctl HTTP 413 weird behavior

- with -d reports correctly the 413, w/o it, it reports the timeout before.

## curl commands

### upload

```
curl -X POST localhost:8080/api/putfile -F "upload[]=@xxx" -F "upload[]=@yyy" -H "Content-Type: multipart/form-data"
```

### download
```
curl -O http://localhost:8080/api/v1/file/388f41f4-3f0d-41e1-a022-9132c0e9e16f/2023-02-28-18-33-xxx
```

### delete
```
curl -X DELETE http://localhost:8080/api/v1/file/388f41f4-3f0d-41e1-a022-9132c0e9e16f/
curl -X DELETE http://localhost:8080/api/v1/file/?id=388f41f4-3f0d-41e1-a022-9132c0e9e16f/
curl -X DELETE -H "Accept: application/json"  -d '{"id":"$id"}' http://localhost:8080/api/v1/file/
```
