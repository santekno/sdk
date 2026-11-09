package s3

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

type suiteVector struct {
	name, method, path, rawQuery string
	headers                      []pair
	body                         string
	signBody                     bool
	headerCreq, headerSig        string
	queryCreq, querySig          string
}

func parseRawQuery(t *testing.T, raw string) []pair {
	t.Helper()
	if raw == "" {
		return nil
	}
	var out []pair
	for _, part := range strings.Split(raw, "&") {
		k, v, _ := strings.Cut(part, "=")
		dk, err := url.PathUnescape(k)
		if err != nil {
			t.Fatal(err)
		}
		dv, err := url.PathUnescape(v)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, pair{dk, dv})
	}
	return out
}

// TestSigV4Suite checks the canonical request and the signature, in header
// and query mode, against the official AWS SigV4 test suite.
func TestSigV4Suite(t *testing.T) {
	s := signer{
		accessKeyID:     "AKIDEXAMPLE",
		secretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		region:          "us-east-1",
		service:         "service",
	}
	ts := time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)
	if len(suiteVectors) < 25 {
		t.Fatalf("only %d vectors", len(suiteVectors))
	}
	for _, v := range suiteVectors {
		t.Run(v.name, func(t *testing.T) {
			path := uriEncode(v.path, true)
			query := parseRawQuery(t, v.rawQuery)
			payload := sha256Hex([]byte(v.body))

			hdrs := append(append([]pair{}, v.headers...), pair{"X-Amz-Date", ts.Format(amzDateFormat)})
			if v.signBody {
				hdrs = append(hdrs, pair{"X-Amz-Content-Sha256", payload})
			}
			creq, _ := canonicalRequest(v.method, path, query, hdrs, payload)
			if creq != v.headerCreq {
				t.Errorf("header canonical request:\n%s\nwant:\n%s", creq, v.headerCreq)
			}
			auth := s.authorization(v.method, path, query, hdrs, payload, ts)
			if !strings.HasSuffix(auth, ", Signature="+v.headerSig) {
				t.Errorf("header signature: %s, want %s", auth, v.headerSig)
			}
			if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/service/aws4_request, SignedHeaders=") {
				t.Errorf("authorization = %s", auth)
			}

			params := s.presignQuery(v.method, path, query, v.headers, payload, ts, time.Hour)
			last := params[len(params)-1]
			if last.k != "X-Amz-Signature" || last.v != v.querySig {
				t.Errorf("query signature = %v, want %s", last, v.querySig)
			}
			qcreq, _ := canonicalRequest(v.method, path, params[:len(params)-1], v.headers, payload)
			if qcreq != v.queryCreq {
				t.Errorf("query canonical request:\n%s\nwant:\n%s", qcreq, v.queryCreq)
			}
		})
	}
}

// S3 examples from the Amazon S3 API reference ("Signature calculations for
// the Authorization header: transferring payload in a single chunk" and
// "Authenticating requests: using query parameters"), credentials
// AKIAIOSFODNN7EXAMPLE / wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY, us-east-1,
// 2013-05-24T00:00:00Z, virtual-hosted host examplebucket.s3.amazonaws.com.
var s3DocSigner = signer{
	accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
	secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	region:          "us-east-1",
	service:         "s3",
}

var s3DocTime = time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)

func TestSigV4S3DocExamples(t *testing.T) {
	const host = "examplebucket.s3.amazonaws.com"
	date := s3DocTime.Format(amzDateFormat)
	putBody := "Welcome to Amazon S3."
	cases := []struct {
		name, method, path string
		query              []pair
		headers            []pair
		payload            string
		wantSigned, want   string
	}{
		{
			name: "GET object", method: "GET", path: "/test.txt",
			headers: []pair{{"Host", host}, {"Range", "bytes=0-9"}, {"x-amz-content-sha256", emptyBodyHash}, {"x-amz-date", date}},
			payload: emptyBodyHash, wantSigned: "host;range;x-amz-content-sha256;x-amz-date",
			want: "f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41",
		},
		{
			name: "PUT object", method: "PUT", path: "/test$file.text",
			headers: []pair{
				{"Host", host}, {"Date", "Fri, 24 May 2013 00:00:00 GMT"}, {"x-amz-date", date},
				{"x-amz-storage-class", "REDUCED_REDUNDANCY"}, {"x-amz-content-sha256", sha256Hex([]byte(putBody))},
			},
			payload: sha256Hex([]byte(putBody)), wantSigned: "date;host;x-amz-content-sha256;x-amz-date;x-amz-storage-class",
			want: "98ad721746da40c64f1a55b78f14c238d841ea1380cd77a1b5971af0ece108bd",
		},
		{
			name: "GET bucket lifecycle", method: "GET", path: "/", query: []pair{{"lifecycle", ""}},
			headers: []pair{{"Host", host}, {"x-amz-content-sha256", emptyBodyHash}, {"x-amz-date", date}},
			payload: emptyBodyHash, wantSigned: "host;x-amz-content-sha256;x-amz-date",
			want: "fea454ca298b7da1c68078a5d1bdbfbbe0d65c699e0f91ac7a200a0136783543",
		},
		{
			name: "GET bucket list objects", method: "GET", path: "/", query: []pair{{"max-keys", "2"}, {"prefix", "J"}},
			headers: []pair{{"Host", host}, {"x-amz-content-sha256", emptyBodyHash}, {"x-amz-date", date}},
			payload: emptyBodyHash, wantSigned: "host;x-amz-content-sha256;x-amz-date",
			want: "34b48302e7b5fa45bde8084f4b7868a86f0a534bc59db6670ed5711ef69dc6f7",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s3DocSigner.authorization(c.method, uriEncode(c.path, true), c.query, c.headers, c.payload, s3DocTime)
			want := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, SignedHeaders=" +
				c.wantSigned + ", Signature=" + c.want
			if got != want {
				t.Errorf("Authorization:\n got %s\nwant %s", got, want)
			}
		})
	}
}

func TestSigV4S3DocPresignExample(t *testing.T) {
	params := s3DocSigner.presignQuery("GET", "/test.txt", nil,
		[]pair{{"Host", "examplebucket.s3.amazonaws.com"}}, unsignedBody, s3DocTime, 86400*time.Second)
	want := []pair{
		{"X-Amz-Algorithm", "AWS4-HMAC-SHA256"},
		{"X-Amz-Credential", "AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request"},
		{"X-Amz-Date", "20130524T000000Z"},
		{"X-Amz-Expires", "86400"},
		{"X-Amz-SignedHeaders", "host"},
		{"X-Amz-Signature", "aeeed9bbccd4d02ee5c0109b86d86835f995330da4c265957d157751f604d404"},
	}
	if len(params) != len(want) {
		t.Fatalf("params = %v", params)
	}
	for i := range want {
		if params[i] != want[i] {
			t.Errorf("param %d = %v, want %v", i, params[i], want[i])
		}
	}
}

func TestURIEncode(t *testing.T) {
	cases := []struct {
		in        string
		keepSlash bool
		want      string
	}{
		{"a/b c", true, "a/b%20c"},
		{"a/b c", false, "a%2Fb%20c"},
		{"-._~AZaz09", false, "-._~AZaz09"},
		{"+*!'()$&=?#%", true, "%2B%2A%21%27%28%29%24%26%3D%3F%23%25"},
		{"ሴ", true, "%E1%88%B4"},
	}
	for _, c := range cases {
		if got := uriEncode(c.in, c.keepSlash); got != c.want {
			t.Errorf("uriEncode(%q, %v) = %q, want %q", c.in, c.keepSlash, got, c.want)
		}
	}
}

func TestCanonicalQueryOrder(t *testing.T) {
	got := canonicalQuery([]pair{{"b", "1"}, {"a", "2"}, {"a", "1"}, {"A", "z"}, {"a b", "c d"}})
	if want := "A=z&a=1&a=2&a%20b=c%20d&b=1"; got != want {
		t.Errorf("canonicalQuery = %q, want %q", got, want)
	}
}
