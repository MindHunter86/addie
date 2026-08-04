// Bash brace expansion tests
//
// * NOTICE
// * This file contains code initially generated with the assistance of AI tooling (this file only not the project as a whole).
// * The resulting implementation has been reviewed, tested, and, where necessary,
// * corrected by the project maintainer prior to publication.

package braceexp

import (
	"slices"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		limit   int
		want    []string
	}{
		{
			name:    "plain string",
			pattern: "cache.example.com",
			want: []string{
				"cache.example.com",
			},
		},
		{
			name:    "ascending range",
			pattern: "cache{1..3}.example.com",
			want: []string{
				"cache1.example.com",
				"cache2.example.com",
				"cache3.example.com",
			},
		},
		{
			name:    "descending range",
			pattern: "cache{3..1}.example.com",
			want: []string{
				"cache3.example.com",
				"cache2.example.com",
				"cache1.example.com",
			},
		},
		{
			name:    "ascending range with step",
			pattern: "cache{1..6..2}.example.com",
			want: []string{
				"cache1.example.com",
				"cache3.example.com",
				"cache5.example.com",
			},
		},
		{
			name:    "descending range with step",
			pattern: "cache{6..1..2}.example.com",
			want: []string{
				"cache6.example.com",
				"cache4.example.com",
				"cache2.example.com",
			},
		},
		{
			name:    "zero padded range",
			pattern: "cache{01..03}.example.com",
			want: []string{
				"cache01.example.com",
				"cache02.example.com",
				"cache03.example.com",
			},
		},
		{
			name:    "negative padded range",
			pattern: "cache{-03..1}.example.com",
			want: []string{
				"cache-03.example.com",
				"cache-02.example.com",
				"cache-01.example.com",
				"cache000.example.com",
				"cache001.example.com",
			},
		},
		{
			name:    "string alternatives",
			pattern: "{cache,api,db}.example.com",
			want: []string{
				"cache.example.com",
				"api.example.com",
				"db.example.com",
			},
		},
		{
			name:    "multiple expressions",
			pattern: "{cache,api}{01..02}.example.com",
			want: []string{
				"cache01.example.com",
				"cache02.example.com",
				"api01.example.com",
				"api02.example.com",
			},
		},
		{
			name:    "range-like string in alternatives",
			pattern: "{1..3,x}.example.com",
			want: []string{
				"1..3.example.com",
				"x.example.com",
			},
		},
		{
			name:    "empty alternative",
			pattern: "{cache,}.example.com",
			want: []string{
				"cache.example.com",
				".example.com",
			},
		},
		{
			name:    "default limit",
			pattern: "cache{1..3}.example.com",
			limit:   -1,
			want: []string{
				"cache1.example.com",
				"cache2.example.com",
				"cache3.example.com",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Expand(tt.pattern, tt.limit)
			if err != nil {
				t.Fatalf(
					"Expand(%q, %d): %v",
					tt.pattern,
					tt.limit,
					err,
				)
			}

			if !slices.Equal(got, tt.want) {
				t.Fatalf(
					"Expand(%q, %d) mismatch\ngot:  %q\nwant: %q",
					tt.pattern,
					tt.limit,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestExpandErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		limit   int
		wantErr string
	}{
		{
			name:    "unclosed brace",
			pattern: "cache{1..3.example.com",
			wantErr: "unclosed brace",
		},
		{
			name:    "nested braces",
			pattern: "{cache{1..2},api}.example.com",
			wantErr: "nested braces are not supported",
		},
		{
			name:    "invalid expression",
			pattern: "cache{abc}.example.com",
			wantErr: "invalid brace expression",
		},
		{
			name:    "invalid range start",
			pattern: "cache{x..3}.example.com",
			wantErr: "invalid range",
		},
		{
			name:    "invalid range end",
			pattern: "cache{1..x}.example.com",
			wantErr: "invalid range",
		},
		{
			name:    "zero step",
			pattern: "cache{1..3..0}.example.com",
			wantErr: "invalid range",
		},
		{
			name:    "missing step",
			pattern: "cache{1..3..}.example.com",
			wantErr: "invalid range",
		},
		{
			name:    "too many separators",
			pattern: "cache{1..3..2..1}.example.com",
			wantErr: "invalid range",
		},
		{
			name:    "limit exceeded",
			pattern: "cache{1..10}.example.com",
			limit:   5,
			wantErr: "expansion exceeds limit 5",
		},
		{
			name:    "product limit exceeded",
			pattern: "{cache,api}{1..10}.example.com",
			limit:   19,
			wantErr: "expansion exceeds limit 19",
		},
		{
			name:    "full int64 range overflows variant count",
			pattern: "{-9223372036854775808..9223372036854775807}",
			wantErr: "expansion is too large",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Expand(tt.pattern, tt.limit)
			if err == nil {
				t.Fatalf(
					"Expand(%q, %d) returned no error",
					tt.pattern,
					tt.limit,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf(
					"Expand(%q, %d) error = %q, want substring %q",
					tt.pattern,
					tt.limit,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

var benchmarkResult []string

func BenchmarkExpand(b *testing.B) {
	benchmarks := []struct {
		name    string
		pattern string
		limit   int
	}{
		{
			name:    "plain",
			pattern: "cache.example.com",
			limit:   1,
		},
		{
			name:    "range_10",
			pattern: "cache{1..10}.example.com",
			limit:   10,
		},
		{
			name:    "range_100",
			pattern: "cache{001..100}.example.com",
			limit:   100,
		},
		{
			name:    "range_1000",
			pattern: "cache{0001..1000}.example.com",
			limit:   1000,
		},
		{
			name:    "alternatives_5",
			pattern: "{cache,api,db,queue,storage}.example.com",
			limit:   5,
		},
		{
			name:    "product_20",
			pattern: "{cache,api}{01..10}.example.com",
			limit:   20,
		},
		{
			name:    "product_100",
			pattern: "{cache,api,db,queue}{01..25}.example.com",
			limit:   100,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()

			var result []string

			for i := 0; i < b.N; i++ {
				var err error

				result, err = Expand(bm.pattern, bm.limit)
				if err != nil {
					b.Fatal(err)
				}
			}

			benchmarkResult = result
		})
	}
}
