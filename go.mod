module git.sr.ht/~dvko/gozer

go 1.23

toolchain go1.24.4

require github.com/yuin/goldmark v1.6.0

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/fsnotify/fsnotify v1.7.0
)

require (
	github.com/sivukhin/godjot/v2 v2.0.1-0.20250612185934-f0b56981998c // indirect
	golang.org/x/sys v0.4.0 // indirect
)

replace github.com/sivukhin/godjot/v2 => ../godjot
