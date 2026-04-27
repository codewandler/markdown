module github.com/codewandler/markdown/examples/stream-readme

go 1.26.1

require (
	github.com/codewandler/markdown v0.0.0
	github.com/codewandler/markdown/adapters/chroma v0.0.0
)

require (
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
)

replace github.com/codewandler/markdown => ../..

replace github.com/codewandler/markdown/adapters/chroma => ../../adapters/chroma
