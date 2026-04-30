module github.com/codewandler/markdown/cmd/mdview

go 1.26.1

replace github.com/codewandler/markdown => ../..

require (
	github.com/codewandler/markdown v0.0.0-00010101000000-000000000000
	github.com/dolmen-go/kittyimg v1.0.0
	golang.org/x/image v0.20.0
)

require (
	github.com/alecthomas/chroma/v2 v2.23.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
)
