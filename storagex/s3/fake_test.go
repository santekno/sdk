package s3

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/santekno/sdk/storagex"
)

const (
	testAKID   = "AKIATESTFAKEKEY0001"
	testSecret = "s3cr3t/TEST+key=VALUE0123456789abcdefXYZ"
	testRegion = "local"
	testBucket = "media-uploads"
	testBase   = "/storage/v1/s3"
)

var testTime = time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)

type fakeObject struct {
	body        []byte
	contentType string
	modTime     time.Time
}

// fakeS3 is a minimal path-style S3 server that verifies SigV4 header and
// query signatures from the raw request as received on the wire.
type fakeS3 struct {
	t       *testing.T
	mu      sync.Mutex
	objects map[string]fakeObject
	now     time.Time
	// override, when set, answers every request instead of the store.
	override func(w http.ResponseWriter, r *http.Request) bool
	// honorIfNoneMatch makes PUT with If-None-Match: * return 412 on existing
	// objects.
	honorIfNoneMatch bool
	requests         []string // method + raw request URI
	headers          []http.Header
	listPageCap      int // forces smaller pages than max-keys when > 0
}

func newFake(t *testing.T) (*fakeS3, *httptest.Server) {
	f := &fakeS3{t: t, objects: map[string]fakeObject{}, now: testTime, honorIfNoneMatch: true}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func newTestClient(t *testing.T, srv *httptest.Server, opts ...storagex.Option) *Client {
	t.Helper()
	c, err := New(Config{
		Endpoint:        srv.URL + testBase,
		Region:          testRegion,
		Bucket:          testBucket,
		AccessKeyID:     testAKID,
		SecretAccessKey: testSecret,
		HTTPClient:      srv.Client(),
		TempDir:         t.TempDir(),
	}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	c.now = func() time.Time { return testTime }
	return c
}

func s3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprintf(w, "<Error><Code>%s</Code><Message>message echoing the key and %s</Message></Error>", code, testAKID)
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.RequestURI)
	f.headers = append(f.headers, r.Header.Clone())
	override := f.override
	f.mu.Unlock()
	if override != nil && override(w, r) {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s3Error(w, 400, "IncompleteBody")
		return
	}
	if code := f.verify(r, body); code != "" {
		s3Error(w, http.StatusForbidden, code)
		return
	}
	rawPath, _, _ := strings.Cut(r.RequestURI, "?")
	bucketPath := testBase + "/" + testBucket
	switch {
	case rawPath == bucketPath && r.Method == http.MethodGet:
		f.list(w, r)
	case strings.HasPrefix(rawPath, bucketPath+"/"):
		key, err := url.PathUnescape(strings.TrimPrefix(rawPath, bucketPath+"/"))
		if err != nil {
			s3Error(w, 400, "InvalidURI")
			return
		}
		f.object(w, r, key, body)
	default:
		s3Error(w, 404, "NoSuchBucket")
	}
}

func (f *fakeS3) object(w http.ResponseWriter, r *http.Request, key string, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := f.objects[key]
	switch r.Method {
	case http.MethodPut:
		if ok && f.honorIfNoneMatch && r.Header.Get("If-None-Match") == "*" {
			s3Error(w, http.StatusPreconditionFailed, "PreconditionFailed")
			return
		}
		f.objects[key] = fakeObject{body: body, contentType: r.Header.Get("Content-Type"), modTime: f.now}
		w.Header().Set("ETag", `"`+fmt.Sprintf("%x", sha256.Sum256(body))[:32]+`"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet, http.MethodHead:
		if !ok {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			s3Error(w, http.StatusNotFound, "NoSuchKey")
			return
		}
		w.Header().Set("Content-Type", obj.contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(obj.body)))
		w.Header().Set("ETag", `"`+fmt.Sprintf("%x", sha256.Sum256(obj.body))[:32]+`"`)
		w.Header().Set("Last-Modified", obj.modTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(obj.body)
		}
	case http.MethodDelete:
		delete(f.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		s3Error(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
	}
}

type fakeListResult struct {
	XMLName               xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
	Name                  string   `xml:"Name"`
	Prefix                string   `xml:"Prefix"`
	KeyCount              int      `xml:"KeyCount"`
	MaxKeys               int      `xml:"MaxKeys"`
	IsTruncated           bool     `xml:"IsTruncated"`
	ContinuationToken     string   `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string   `xml:"NextContinuationToken,omitempty"`
	Contents              []fakeListEntry
}

type fakeListEntry struct {
	XMLName      xml.Name `xml:"Contents"`
	Key          string   `xml:"Key"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
	Size         int      `xml:"Size"`
	StorageClass string   `xml:"StorageClass"`
}

func (f *fakeS3) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("list-type") != "2" {
		s3Error(w, 400, "InvalidArgument")
		return
	}
	maxKeys, _ := strconv.Atoi(q.Get("max-keys"))
	if f.listPageCap > 0 && f.listPageCap < maxKeys {
		maxKeys = f.listPageCap
	}
	after := ""
	if tok := q.Get("continuation-token"); tok != "" {
		b, err := hex.DecodeString(tok)
		if err != nil {
			s3Error(w, 400, "InvalidArgument")
			return
		}
		after = string(b)
	}
	f.mu.Lock()
	var keys []string
	for k := range f.objects {
		if strings.HasPrefix(k, q.Get("prefix")) && k > after {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	res := fakeListResult{Name: testBucket, Prefix: q.Get("prefix"), MaxKeys: maxKeys, ContinuationToken: q.Get("continuation-token")}
	for i, k := range keys {
		if i == maxKeys {
			res.IsTruncated = true
			res.NextContinuationToken = hex.EncodeToString([]byte(keys[i-1]))
			break
		}
		o := f.objects[k]
		res.Contents = append(res.Contents, fakeListEntry{
			Key: k, LastModified: o.modTime.Format("2006-01-02T15:04:05.000Z"),
			ETag: `"etag-` + k + `"`, Size: len(o.body), StorageClass: "STANDARD",
		})
	}
	f.mu.Unlock()
	res.KeyCount = len(res.Contents)
	w.Header().Set("Content-Type", "application/xml")
	_, _ = io.WriteString(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(res)
}

// sigParts are the signature inputs taken from a request.
type sigParts struct {
	credential, signedHeaders, signature, amzDate, payloadHash string
}

// rawQueryPairs decodes the raw query, leaving out X-Amz-Signature.
func rawQueryPairs(rawQuery string) ([]pair, map[string]string, bool) {
	var query []pair
	params := map[string]string{}
	if rawQuery == "" {
		return nil, params, true
	}
	for _, part := range strings.Split(rawQuery, "&") {
		k, v, _ := strings.Cut(part, "=")
		dk, err1 := url.PathUnescape(k)
		dv, err2 := url.PathUnescape(v)
		if err1 != nil || err2 != nil {
			return nil, nil, false
		}
		params[dk] = dv
		if dk != "X-Amz-Signature" {
			query = append(query, pair{dk, dv})
		}
	}
	return query, params, true
}

// queryAuth reads presigned parameters and rejects expired URLs.
func (f *fakeS3) queryAuth(params map[string]string) (sigParts, string) {
	p := sigParts{params["X-Amz-Credential"], params["X-Amz-SignedHeaders"], params["X-Amz-Signature"], params["X-Amz-Date"], unsignedBody}
	exp, err := strconv.Atoi(params["X-Amz-Expires"])
	ts, terr := time.Parse(amzDateFormat, p.amzDate)
	if err != nil || terr != nil || f.now.After(ts.Add(time.Duration(exp)*time.Second)) {
		return p, "AccessDenied" // expired
	}
	return p, ""
}

// headerAuth reads the Authorization header and checks the body hash.
func headerAuth(r *http.Request, body []byte) (sigParts, string) {
	var p sigParts
	rest, ok := strings.CutPrefix(r.Header.Get("Authorization"), sigAlgorithm+" ")
	if !ok {
		return p, "AccessDenied"
	}
	for _, part := range strings.Split(rest, ", ") {
		k, v, _ := strings.Cut(part, "=")
		switch k {
		case "Credential":
			p.credential = v
		case "SignedHeaders":
			p.signedHeaders = v
		case "Signature":
			p.signature = v
		}
	}
	p.amzDate, p.payloadHash = r.Header.Get("X-Amz-Date"), r.Header.Get("X-Amz-Content-Sha256")
	if p.payloadHash != unsignedBody && p.payloadHash != fmt.Sprintf("%x", sha256.Sum256(body)) {
		return p, "XAmzContentSHA256Mismatch"
	}
	return p, ""
}

// signedHeaderPairs collects the signed headers as received on the wire.
func signedHeaderPairs(r *http.Request, signed string) ([]pair, bool) {
	var headers []pair
	for _, name := range strings.Split(signed, ";") {
		if name == "host" {
			headers = append(headers, pair{"host", r.Host})
			continue
		}
		vals, ok := r.Header[http.CanonicalHeaderKey(name)]
		if !ok {
			return nil, false // signed header missing on the wire
		}
		for _, v := range vals {
			headers = append(headers, pair{name, v})
		}
	}
	return headers, true
}

// verify checks the SigV4 signature (header or query) and returns an S3
// error code, or "" when the request is authentic.
func (f *fakeS3) verify(r *http.Request, body []byte) string {
	rawPath, rawQuery, _ := strings.Cut(r.RequestURI, "?")
	query, params, ok := rawQueryPairs(rawQuery)
	if !ok {
		return "InvalidArgument"
	}
	var p sigParts
	var code string
	if params["X-Amz-Algorithm"] != "" {
		p, code = f.queryAuth(params)
	} else {
		p, code = headerAuth(r, body)
	}
	if code != "" {
		return code
	}
	ts, err := time.Parse(amzDateFormat, p.amzDate)
	if err != nil {
		return "AccessDenied"
	}
	s := signer{accessKeyID: testAKID, secretAccessKey: testSecret, region: testRegion, service: "s3"}
	if p.credential != testAKID+"/"+s.scope(ts) {
		return "InvalidAccessKeyId"
	}
	headers, ok := signedHeaderPairs(r, p.signedHeaders)
	if !ok {
		return "AccessDenied"
	}
	creq, gotSigned := canonicalRequest(r.Method, rawPath, query, headers, p.payloadHash)
	if gotSigned != p.signedHeaders || s.signature(ts, s.stringToSign(ts, creq)) != p.signature {
		return "SignatureDoesNotMatch"
	}
	return ""
}
