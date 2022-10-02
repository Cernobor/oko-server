package server

const (
	URIPing              = "/ping"
	URIBuildInfo         = "/build-info"
	URIHardFail          = "/hard-fail"
	URISoftFail          = "/soft-fail"
	URIAppVersions       = "/app-versions"
	URIAppVersion        = "/app-versions/:version"
	URIReinit            = "/reinit"
	URIMapPack           = "/mappack"
	URIHandshake         = "/handshake"
	URIData              = "/data"
	URIDataPeople        = "/data/people"
	URIDataFeatures      = "/data/features"
	URIDataFeaturesPhoto = "/data/features/:feature/photos/:photo"
	URIDataProposals     = "/data/proposals"
	URITileserverRoot    = "/tileserver"
	URITileserver        = URITileserverRoot + "/*x"
	URITileTemplate      = URITileserverRoot + "/map/tiles/{z}/{x}/{y}.pbf"

	AppName = "OKO"
)
