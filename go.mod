module github.com/petems/whisper-tray

go 1.22

require (
	github.com/atotto/clipboard v0.1.4
	github.com/getlantern/systray v1.2.2
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20240101000000-000000000000
	github.com/gordonklaus/portaudio v0.0.0-20230709114228-aafa478834f5
	github.com/rs/zerolog v1.32.0
)

// Use vendored whisper.cpp bindings
replace github.com/ggerganov/whisper.cpp/bindings/go => ./vendor/whisper.cpp/bindings/go

require (
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520 // indirect
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7 // indirect
	github.com/getlantern/golog v0.0.0-20190830074920-4ef2e798c2d7 // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55 // indirect
	github.com/getlantern/ops v0.0.0-20190325191751-d70cb0d6f85f // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	golang.org/x/sys v0.12.0 // indirect
)
