package version

// Version is set at build time via:
//
//	go build -ldflags="-X labelsrv/internal/version.Version=1.2.3"
var Version = "dev"
