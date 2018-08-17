package wok

import (
	"context"
	"net/http"
	"strconv"

	"github.com/manvalls/wit"
)

// Deduper runs steps if they were not ran before
type Deduper struct {
	header string
	keys   map[uint]bool
	scope  *Scope
}

// NewDeduper builds a new deduper linked to this scope
func (s *Scope) NewDeduper(header string) Deduper {
	header = http.CanonicalHeaderKey(header)
	_, keyList := s.FromHeader(header)

	keys := map[uint]bool{}
	for _, key := range keyList {
		keys[key] = true
	}

	return Deduper{header, keys, s}
}

// Dedupe runs the provided function if necessary
func (d Deduper) Dedupe(key uint, f func(ctx context.Context) wit.Delta) wit.Delta {
	if !d.keys[key] {
		delta := wit.Run(d.scope.req.Context(), f)
		h := strconv.Quote(d.header)
		k := strconv.Quote(strconv.FormatUint(uint64(key), 36))

		return wit.List(
			wit.Head.One(wit.Append(wit.FromString(
				"<script>(function(){var w=window.wok=window.wok||{},h="+h+",k="+k+",a='dedupes',d=w[a]=w[a]||{},l=(d[h]?d[h].split(','):[]);if(l.indexOf(k)==-1)l.push(k);if(l.length)d[h]=l.join()})()</script>",
			))),
			delta,
		)
	}

	return wit.Nil
}

// Reset marks a given key as unused
func (d Deduper) Reset(key uint) wit.Delta {
	h := strconv.Quote(d.header)
	k := strconv.Quote(strconv.FormatUint(uint64(key), 36))

	return wit.Head.One(wit.Append(wit.FromString(
		"<script>(function(){var w=window.wok=window.wok||{},h=" + h + ",k=" + k + ",a='dedupes',d=w[a]=w[a]||{},l=(d[h]?d[h].split(','):[]),i=l.indexOf(k);if(i!=-1)l.splice(i,1);if(l.length)d[h]=l.join();else delete d[h]})()</script>",
	)))
}
