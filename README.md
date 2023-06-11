<p align="center">
  <a href="" rel="noopener">
  <img src="https://raw.githubusercontent.com/Cernobor/oko-app/master/assets/splash.png" alt="Project logo" style="background: #153d24;"></a>
</p>

<h3 align="center">Organizátorské Kartografické OKO - server</h3>

<div align="center">

[![Status](https://img.shields.io/badge/status-active-success.svg)]()
[![GitHub Issues](https://img.shields.io/github/issues/Cernobor/oko-server.svg)](https://github.com/Cernobor/oko-server/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/Cernobor/oko-server.svg)](https://github.com/Cernobor/oko-server/pulls)
<!-- [![License](https://img.shields.io/badge/license-MIT-blue.svg)](/LICENSE) -->

</div>

---

## 📝 Table of Contents

- [About](#about)
- [Building](#building)
- [Usage](#usage)
- [Built Using](#built_using)
- [Authors](#authors)
- [Acknowledgments](#acknowledgement)

## 🧐 About <a name = "about"></a>

Central server for the [OKO](https://github.com/Cernobor/oko-app) app.
Manges users, provides map tiles and the whole map tile pack, receives geo features submitted by users and serves them back.

## 🏗 Building <a name = "building"></a>

The only requirement to build the server is [go](https://go.dev).

In the root of the project, run

```
go build
```

which will produce the executable ``oko-server`` which you can just run (see [Usage](#usage)).

## 🎈 Usage <a name="usage"></a>

### Using docker

Dockerfile is included, therefore to build the docker image, all that is needed to do is to run

```
docker build -t name:tag /path/to/the/root/of/the/repository
```
or, if you have no local changes, you can build the image even without cloning the repository at all
```
docker build -t name:tag https://github.com/Cernobor/oko-server.git
```

The image can then be run using the ``name:tag`` that you have given in the previous step.
Volume ``/data`` is available to share data between the host and the container.

```
docker run -v /path/to/data:/data name:tag [options...]
```

### Without docker

Just run the previously built (see [Building](#building)) executable
```
/path/to/oko-server [options...]
```

### Options

The server is configurable via a set of options, some of which are required.
The table of options follows:

| option | description |
| ------ | ----------- |
| ``-tilepack <path>`` | **Required.** File that will be sent to clients when they request a tile pack, also used to serve tiles in online mode. |
| ``-port <port>`` | Port where the server will listen. Default is ``8080``. |
| ``-dbfile <path>`` | Path to the sqlite3 database file used for data storage. Will be created, if it does not exist upons startup. Default is ``./data.sqlite3``. |
| ``-min-zoom <int>`` | Minimum supported zoom. Clients will receive this value during handshake. Default is ``1``. |
| ``-default-center-lat <float>`` | Latitude of the default map center, formatted as a floating point number of degrees. Clients will receive this value during handhake. Default is ``0``. |
| ``-default-center-lng <float>`` | Longitude of the default map center, formatted as a floating point number of degrees. Clients will receive this value during handhake. Default is ``0``. |
| ``-max-photo-width <int>`` | Maximum width of stored pohotos. Uploaded photos exceeding this width will be rescaled to fit (keeping aspect ratio). If ``0``, no scaling will be done. Default is ``0``. |
| ``-max-photo-height <int>`` | Maximum height of stored pohotos. Uploaded photos exceeding this height will be rescaled to fit (keeping aspect ratio). If ``0``, no scaling will be done. Default is ``0``. |
| ``-photo-quality <int>`` | JPEG photo quality. Photos are reencoded to JPEG with this quality setting when uploaded. Default is ``90``. |
| ``-debug`` | If specified, logging level will be set do debug instead of info. w

## ⛏️ Built Using <a name = "built_using"></a>

- [Go](https://go.dev/) - Programming language
- [Gin](https://gin-gonic.com/) - HTTP handler
- [SQLite](https://www.sqlite.org/) - Database
- [mbtileserver](https://github.com/consbio/mbtileserver) - MBTiles tile server

## ✍️ Authors <a name = "authors"></a>

- [@zegkljan](https://github.com/zegkljan) - Idea & Initial work
- [@0x416E64](https://github.com/0x416E64) - Docker and deployment tweaks

See also the list of [contributors](https://github.com/Cernobor/oko-server/contributors) who participated in this project.

## 🎉 Acknowledgements <a name = "acknowledgement"></a>

- Hat tip to anyone whose code was used
