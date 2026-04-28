module github.com/codewandler/markdown/examples/stream-readme

go 1.26.1

require (
	github.com/codewandler/markdown v0.0.0
	github.com/codewandler/markdown/adapters/chroma v0.0.0
)

replace github.com/codewandler/markdown => ../..

replace github.com/codewandler/markdown/adapters/chroma => ../../adapters/chroma
