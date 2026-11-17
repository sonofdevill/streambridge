module streambridge

go 1.24.7

replace golang.org/x/net => github.com/golang/net v0.27.0

replace golang.org/x/sys => github.com/golang/sys v0.22.0

replace golang.org/x/sync => github.com/golang/sync v0.7.0

replace golang.org/x/tools => github.com/golang/tools v0.22.0

replace golang.org/x/mod => github.com/golang/mod v0.18.0

require (
	github.com/datarhei/joy4 v0.0.0-20261106082943-f1b6cc8614bb
	github.com/miekg/dns v1.1.62
)

require (
	fyne.io/systray v1.11.0
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gorilla/websocket v1.5.3
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
)

replace fyne.io/systray => github.com/fyne-io/systray v1.11.0
