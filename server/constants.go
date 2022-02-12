package server

const (
	URIPing              = "/ping"
	URIHardFail          = "/hard-fail"
	URISoftFail          = "/soft-fail"
	URIReinit            = "/reinit"
	URIMapPack           = "/mappack"
	URIHandshake         = "/handshake"
	URIData              = "/data"
	URIDataPeople        = "/data/people"
	URIDataFeatures      = "/data/features"
	URIDataFeaturesPhoto = "/data/features/:feature/photos/:photo"
	URITileserverRoot    = "/tileserver"
	URITileserver        = URITileserverRoot + "/*x"
	URITileTemplate      = URITileserverRoot + "/map/tiles/{z}/{x}/{y}.pbf"
)
