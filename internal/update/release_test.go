package update

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func zipPackage(t *testing.T, exe []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, name := range packageFiles {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		data := []byte("updated " + name)
		if name == packageFiles[0] {
			data = exe
		}
		if _, err = w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func fixture(t *testing.T) (*Client, Release, []byte) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	// A distinct, runnable Windows binary, without building another toolchain.
	data = append(data, []byte("squirrel-update-fixture-v1.2.0")...)
	archive := zipPackage(t, data)
	r := Release{Tag: "v1.2.0", Notes: "Fixture release notes"}
	name := r.archiveName(runtime.GOARCH)
	sumName := "SHA256SUMS-" + runtime.GOARCH + ".txt"
	base := "https://github.com/" + Repository + "/releases/download/" + r.Tag + "/"
	r.Assets = []Asset{{Name: name, URL: base + name, Size: int64(len(archive))}, {Name: sumName, URL: base + sumName}}
	hash := sha256.Sum256(archive)
	sum := hex.EncodeToString(hash[:]) + "  " + name + "\n"
	metadata, _ := json.Marshal(r)
	c := &Client{HTTP: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		var b []byte
		switch req.URL.String() {
		case latestURL:
			b = metadata
		case base + name:
			b = archive
		case base + sumName:
			b = []byte(sum)
		default:
			return nil, errors.New("unexpected URL")
		}
		return &http.Response{StatusCode: 200, ContentLength: int64(len(b)), Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header)}, nil
	})}}
	return c, r, data
}

func TestVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		candidate, current string
		want               bool
	}{{"v1.10.0", "1.9.9", true}, {"v1.2.0", "1.2.0", false}, {"1.1.99", "1.2.0", false}, {"2.0.0", "1.99.99", true}} {
		got, err := Newer(tc.candidate, tc.current)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: %v %v", tc, got, err)
		}
	}
	for _, v := range []string{"", "dev", "v1.2.3-beta", "1.02.3", "999999999999999999999999.0.0"} {
		if _, err := Newer(v, "1.0.0"); err == nil {
			t.Fatal("accepted", v)
		}
	}
}

func TestCheckReleaseValidation(t *testing.T) {
	c, r, _ := fixture(t)
	if found, err := c.Check(context.Background(), "1.0.0", runtime.GOARCH); err != nil || found.Version() != "1.2.0" {
		t.Fatal(found, err)
	}
	if found, err := c.Check(context.Background(), "1.2.0", runtime.GOARCH); err != nil || found != nil {
		t.Fatal(found, err)
	}
	for _, kind := range []string{"draft", "prerelease", "url", "duplicate", "missing"} {
		t.Run(kind, func(t *testing.T) {
			bad := r
			bad.Assets = append([]Asset{}, r.Assets...)
			switch kind {
			case "draft":
				bad.Draft = true
			case "prerelease":
				bad.Prerelease = true
			case "url":
				bad.Assets[0].URL = "https://example.com/file.zip"
			case "duplicate":
				bad.Assets = append(bad.Assets, bad.Assets[0])
			case "missing":
				bad.Assets = nil
			}
			b, _ := json.Marshal(bad)
			client := &Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, ContentLength: int64(len(b)), Body: io.NopCloser(bytes.NewReader(b))}, nil
			})}}
			if _, err := client.Check(context.Background(), "1.0.0", runtime.GOARCH); err == nil {
				t.Fatal("accepted invalid release")
			}
		})
	}
}

func makeTarget(t *testing.T, name string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), name)
	exe, _ := os.Executable()
	if err := copyFile(exe, target); err != nil {
		t.Fatal(err)
	}
	return target
}

func TestDownloadFailureNeverTouchesInstallation(t *testing.T) {
	for _, kind := range []string{"hash", "truncated", "network", "canceled", "wrong-arch"} {
		t.Run(kind, func(t *testing.T) {
			c, r, _ := fixture(t)
			original := c.HTTP.Transport
			c.HTTP.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				res, err := original.RoundTrip(req)
				if err != nil {
					return res, err
				}
				if strings.HasSuffix(req.URL.Path, ".zip") {
					switch kind {
					case "hash":
						b, _ := io.ReadAll(res.Body)
						res.Body.Close()
						b[len(b)/2] ^= 1
						res.Body = io.NopCloser(bytes.NewReader(b))
					case "truncated":
						res.Body.Close()
						res.Body = io.NopCloser(strings.NewReader("partial"))
					case "network":
						res.Body.Close()
						return nil, errors.New("network offline")
					}
				}
				return res, nil
			})
			target := makeTarget(t, "squirrel_translate.exe")
			before, _ := fileHash(target)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "canceled" {
				cancel()
			}
			arch := runtime.GOARCH
			if kind == "wrong-arch" {
				arch = "386"
			}
			if _, err := c.Prepare(ctx, r, target, arch, nil, nil); err == nil {
				t.Fatal("expected download rejection")
			}
			after, _ := fileHash(target)
			if after != before {
				t.Fatal("existing executable changed")
			}
			files, _ := os.ReadDir(filepath.Dir(target))
			if len(files) != 1 {
				t.Fatal("failed download left staging files", files)
			}
		})
	}
}

func TestChecksumAndArchiveRejection(t *testing.T) {
	for _, text := range []string{"bad  a.zip", "", "" + strings.Repeat("a", 64) + "  a.zip\n" + strings.Repeat("b", 64) + "  a.zip"} {
		if _, err := checksum(text, "a.zip"); err == nil {
			t.Fatal("accepted bad checksum")
		}
	}
	c, r, _ := fixture(t)
	target := makeTarget(t, "squirrel_translate.exe")
	p, err := c.Prepare(context.Background(), r, target, runtime.GOARCH, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Verified SHA is necessary but insufficient: paths and PE architecture
	// must be checked too, even when a publisher supplied matching checksums.
	for _, kind := range []string{"path", "duplicate", "architecture"} {
		t.Run(kind, func(t *testing.T) {
			var buf bytes.Buffer
			z := zip.NewWriter(&buf)
			for i, name := range packageFiles {
				if i == 0 && kind == "path" {
					name = "../escape.exe"
				}
				if i == 1 && kind == "duplicate" {
					name = packageFiles[0]
				}
				w, _ := z.Create(name)
				w.Write([]byte("not a PE file"))
			}
			z.Close()
			sum := sha256.Sum256(buf.Bytes())
			p.Hash = hex.EncodeToString(sum[:])
			os.WriteFile(filepath.Join(p.Work, "package.zip"), buf.Bytes(), 0600)
			if err := p.unpack(); err == nil {
				t.Fatal("accepted unsafe archive")
			}
		})
	}
}

func TestPublishedReleaseDownload(t *testing.T) {
	if os.Getenv("SQUIRREL_TEST_GITHUB") != "1" {
		t.Skip("opt in to live GitHub download with SQUIRREL_TEST_GITHUB=1")
	}
	c := NewClient()
	r, err := c.Check(context.Background(), "0.0.0", runtime.GOARCH)
	if err != nil || r == nil {
		t.Fatal(r, err)
	}
	p, err := c.Prepare(context.Background(), *r, makeTarget(t, "squirrel_translate.exe"), runtime.GOARCH, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Verified GitHub release %s, %s, SHA256=%s; isolated staging only", r.Tag, p.Architecture, p.Hash)
}
