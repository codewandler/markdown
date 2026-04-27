module github.com/codewandler/markdown/adapters/chroma

go 1.26.1

require (
	github.com/alecthomas/chroma/v2 v2.14.0
	github.com/codewandler/markdown v0.0.0
)

require github.com/dlclark/regexp2 v1.11.0 // indirect

replace github.com/codewandler/markdown => ../..
