package stream

import (
	"reflect"
	"testing"
)

func TestSplitEquivalence(t *testing.T) {
	samples := []string{
		"# Heading\n",
		"alpha\nbeta\n\n",
		"```go\npackage main\n```\n",
		"```go\npackage main\n",
		"- one\n- two\n\n",
		"> quote\n\n",
		"---\n",
		"A **strong** and `code`\n",
		"[foo]:\n/url\n  \"title\"\n\n[foo]\n",
	}
	for _, sample := range samples {
		want := viewEvents(parseAll(t, sample))
		for split := 0; split <= len(sample); split++ {
			t.Run(sampleName(sample, split), func(t *testing.T) {
				p := NewParser()
				var all []Event
				events, err := p.Write([]byte(sample[:split]))
				if err != nil {
					t.Fatal(err)
				}
				all = append(all, events...)
				events, err = p.Write([]byte(sample[split:]))
				if err != nil {
					t.Fatal(err)
				}
				all = append(all, events...)
				events, err = p.Flush()
				if err != nil {
					t.Fatal(err)
				}
				all = append(all, events...)
				if got := viewEvents(all); !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d mismatch\nwant: %#v\n got: %#v", split, want, got)
				}
			})
		}
	}
}

func sampleName(sample string, split int) string {
	if len(sample) > 16 {
		sample = sample[:16]
	}
	return sample + "/" + string(rune('a'+split%26))
}
