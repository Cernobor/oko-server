package server

const (
	URIPing           = "/ping"
	URIHardFail       = "/hard-fail"
	URISoftFail       = "/soft-fail"
	URITilepack       = "/tilepack"
	URIHandshake      = "/handshake"
	URIData           = "/data"
	URIDataPeople     = "/data/people"
	URIDataExtra      = "/data/extra"
	URIDataFeatures   = "/data/features"
	URITileserverRoot = "/tileserver"
	URITileserver     = URITileserverRoot + "/*x"
)
