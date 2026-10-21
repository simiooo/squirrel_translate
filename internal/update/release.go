// Package update downloads published releases and stages verified updates.
// It does not know about translation sessions, credentials or UI windows.
package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const Repository = "simiooo/squirrel_translate"
const latestURL = "https://api.github.com/repos/" + Repository + "/releases/latest"
const maxArchive = 256 << 20

var versionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var packageFiles = []string{"squirrel_translate.exe", "LICENSE", "THIRD_PARTY_NOTICES.txt", "PRIVACY.md"}

func versionParts(s string) ([3]uint64, error) {
	var p [3]uint64
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return p, fmt.Errorf("invalid stable version: %q", s)
	}
	for i := range p {
		v, err := strconv.ParseUint(m[i+1], 10, 64)
		if err != nil {
			return p, err
		}
		p[i] = v
	}
	return p, nil
}

func Newer(candidate, current string) (bool, error) {
	a, err := versionParts(candidate)
	if err != nil {
		return false, err
	}
	b, err := versionParts(current)
	if err != nil {
		return false, err
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i], nil
		}
	}
	return false, nil
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type Release struct {
	Tag        string  `json:"tag_name"`
	Notes      string  `json:"body"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }
func (r Release) asset(name string) (Asset, error) {
	var found []Asset
	for _, a := range r.Assets {
		if a.Name == name {
			found = append(found, a)
		}
	}
	if len(found) != 1 {
		return Asset{}, fmt.Errorf("missing or ambiguous release asset: %s", name)
	}
	a := found[0]
	// Pin the initial URL, including the exact release tag and asset name.
	if a.URL != "https://github.com/"+Repository+"/releases/download/"+r.Tag+"/"+name {
		return Asset{}, errors.New("unexpected release download origin")
	}
	return a, nil
}

type Client struct{ HTTP *http.Client }

func NewClient() *Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 30 * time.Second
	return &Client{HTTP: &http.Client{Transport: t, Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != "https" {
			return errors.New("unsafe download redirect")
		}
		return nil
	}}}
}

func (c *Client) get(ctx context.Context, url string, dst io.Writer, limit int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Squirrel-Translate-Updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub HTTP %d", res.StatusCode)
	}
	if res.ContentLength > limit {
		return errors.New("download exceeds size limit")
	}
	n, err := io.Copy(dst, io.LimitReader(res.Body, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return errors.New("download exceeds size limit")
	}
	if res.ContentLength >= 0 && n != res.ContentLength {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (c *Client) Check(ctx context.Context, current, arch string) (*Release, error) {
	if arch != "amd64" && arch != "arm64" {
		return nil, errors.New("unsupported architecture")
	}
	var data strings.Builder
	if err := c.get(ctx, latestURL, &data, 2<<20); err != nil {
		return nil, err
	}
	var r Release
	if err := json.Unmarshal([]byte(data.String()), &r); err != nil {
		return nil, err
	}
	if r.Draft || r.Prerelease || !strings.HasPrefix(r.Tag, "v") {
		return nil, errors.New("expected a published stable release")
	}
	newer, err := Newer(r.Tag, current)
	if err != nil || !newer {
		return nil, err
	}
	for _, name := range []string{r.archiveName(arch), "SHA256SUMS-" + arch + ".txt"} {
		if _, err := r.asset(name); err != nil {
			return nil, err
		}
	}
	return &r, nil
}
func (r Release) archiveName(arch string) string {
	return "squirrel_translate-" + r.Version() + "-windows-" + arch + ".zip"
}

func checksum(data, name string) (string, error) {
	var found []string
	for _, line := range strings.Split(data, "\n") {
		parts := strings.SplitN(strings.TrimSuffix(line, "\r"), "  ", 2)
		if len(parts) == 2 && parts[1] == name {
			b, err := hex.DecodeString(parts[0])
			if err != nil || len(b) != sha256.Size {
				return "", errors.New("invalid SHA256 checksum")
			}
			found = append(found, strings.ToLower(parts[0]))
		}
	}
	if len(found) != 1 {
		return "", errors.New("missing or ambiguous SHA256 checksum")
	}
	return found[0], nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type progressWriter struct {
	done, total int64
	report      func(int64, int64)
	last        time.Time
}

func (w *progressWriter) Write(b []byte) (int, error) {
	w.done += int64(len(b))
	if w.report != nil && time.Since(w.last) > 150*time.Millisecond {
		w.report(w.done, w.total)
		w.last = time.Now()
	}
	return len(b), nil
}

// Prepare makes a sibling staging directory, proving the actual installation
// is writable before the app exits. It never overwrites installed files.
func (c *Client) Prepare(ctx context.Context, r Release, target, arch string, args []string, progress func(int64, int64)) (_ *Plan, err error) {
	if _, err = versionParts(r.Tag); err != nil {
		return nil, err
	}
	if arch != "amd64" && arch != "arm64" {
		return nil, errors.New("unsupported architecture")
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	resolved, e := filepath.EvalSymlinks(target)
	if e != nil {
		return nil, e
	}
	if !strings.EqualFold(resolved, target) {
		return nil, errors.New("cannot update through a symbolic link")
	}
	work, e := os.MkdirTemp(filepath.Dir(target), ".squirrel-update-")
	if e != nil {
		return nil, fmt.Errorf("installation is not writable: %w", e)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(work)
		}
	}()
	archive, e := r.asset(r.archiveName(arch))
	if e != nil {
		return nil, e
	}
	sums, e := r.asset("SHA256SUMS-" + arch + ".txt")
	if e != nil {
		return nil, e
	}
	var data strings.Builder
	if err = c.get(ctx, sums.URL, &data, 1<<20); err != nil {
		return nil, err
	}
	hash, e := checksum(data.String(), archive.Name)
	if e != nil {
		return nil, e
	}
	f, e := os.OpenFile(filepath.Join(work, "package.zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return nil, e
	}
	pw := &progressWriter{total: archive.Size, report: progress}
	err = c.get(ctx, archive.URL, io.MultiWriter(f, pw), maxArchive)
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if archive.Size <= 0 || pw.done != archive.Size {
		return nil, errors.New("release asset size mismatch")
	}
	p := &Plan{Target: target, Work: work, Version: r.Version(), Architecture: arch, Hash: hash, Args: args, ParentPID: os.Getpid()}
	if err = p.unpack(); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = copyFile(target, filepath.Join(work, "helper.exe")); err != nil {
		return nil, err
	}
	if err = writeJSON(filepath.Join(work, "plan.json"), p); err != nil {
		return nil, err
	}
	if progress != nil {
		progress(pw.done, pw.done)
	}
	return p, nil
}

func (p *Plan) unpack() error {
	hash, err := fileHash(filepath.Join(p.Work, "package.zip"))
	if err != nil {
		return err
	}
	if hash != p.Hash {
		return errors.New("SHA256 verification failed")
	}
	z, err := zip.OpenReader(filepath.Join(p.Work, "package.zip"))
	if err != nil {
		return err
	}
	defer z.Close()
	if len(z.File) != len(packageFiles) {
		return errors.New("unexpected archive contents")
	}
	dir := filepath.Join(p.Work, "new")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, name := range packageFiles {
		var matches []*zip.File
		for _, f := range z.File {
			if f.Name == name {
				matches = append(matches, f)
			}
		}
		if len(matches) != 1 || !matches[0].Mode().IsRegular() || matches[0].UncompressedSize64 > maxArchive {
			return errors.New("invalid archive entry: " + name)
		}
		src, e := matches[0].Open()
		if e != nil {
			return e
		}
		dst, e := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
		if e != nil {
			src.Close()
			return e
		}
		n, e := io.Copy(dst, io.LimitReader(src, maxArchive+1))
		src.Close()
		ce := dst.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		if n > maxArchive {
			return errors.New("expanded archive too large")
		}
	}
	exe, err := pe.Open(filepath.Join(dir, packageFiles[0]))
	if err != nil {
		return fmt.Errorf("invalid Windows executable: %w", err)
	}
	defer exe.Close()
	want := uint16(pe.IMAGE_FILE_MACHINE_AMD64)
	if p.Architecture == "arm64" {
		want = pe.IMAGE_FILE_MACHINE_ARM64
	}
	if exe.Machine != want {
		return errors.New("executable architecture mismatch")
	}
	return nil
}
