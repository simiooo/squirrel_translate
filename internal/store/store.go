package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/simiooo/squirrel_translate/internal/core"
	"golang.org/x/text/unicode/norm"
)

// A versioned JSON store is enough for the prototype and keeps distribution
// small. All updates use a temp file and replacement; no transient window owns
// vocabulary data. The Store interface can later be backed by SQLite.
type Store struct {
	Dir string
	mu  sync.Mutex
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}
func (s *Store) LoadConfig() (core.Config, error) {
	c := core.DefaultConfig()
	b, e := os.ReadFile(filepath.Join(s.Dir, "settings.json"))
	if os.IsNotExist(e) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	return c, e
}
func (s *Store) SaveConfig(c core.Config) error {
	if e := c.Check(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write("settings.json", c)
}

type vocabulary struct {
	Version int          `json:"version"`
	Entries []core.Entry `json:"entries"`
}

func (s *Store) read() ([]core.Entry, error) {
	b, e := os.ReadFile(filepath.Join(s.Dir, "vocabulary.json"))
	if os.IsNotExist(e) {
		return []core.Entry{}, nil
	}
	if e != nil {
		return nil, e
	}
	var v vocabulary
	if e = json.Unmarshal(b, &v); e != nil {
		return nil, fmt.Errorf(i18n.T("词汇表损坏，保留原文件：%w"), e)
	}
	if v.Version != 1 {
		return nil, fmt.Errorf(i18n.T("不支持的词汇表版本 %d"), v.Version)
	}
	return v.Entries, nil
}
func (s *Store) List() ([]core.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.read()
	sort.Slice(e, func(i, j int) bool { return e[i].UpdatedAt.After(e[j].UpdatedAt) })
	return e, err
}
func entryID(word, source, target string) string {
	key := norm.NFC.String(strings.TrimSpace(word)) + "\x00" + strings.ToLower(source) + "\x00" + target
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:12])
}

func Matches(e core.Entry, lemma, source, target string) bool {
	return e.ID == entryID(lemma, source, target)
}

func (s *Store) Save(e core.Entry) (bool, error) {
	if err := core.Validate.Struct(e.Definition); err != nil {
		return false, fmt.Errorf(i18n.T("释义尚不完整，无法收藏：%w"), err)
	}
	if e.Kind == "" {
		e.Word = e.Definition.Lemma
		e.Kind = e.LexicalKind()
	}
	if !core.IsLexical(e.Kind) {
		return false, fmt.Errorf("%s", i18n.T("只有单词与短语可以加入词汇表"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return false, err
	}
	now := time.Now()
	e.ID = entryID(e.Definition.Lemma, e.SourceLanguage, e.TargetLanguage)
	e.Word = e.Definition.Lemma
	e.UpdatedAt = now
	for i, old := range entries {
		if old.ID == e.ID {
			e.AddedAt = old.AddedAt
			if e.Note == "" {
				e.Note = old.Note
			}
			entries[i] = e
			return false, s.write("vocabulary.json", vocabulary{1, entries})
		}
	}
	e.AddedAt = now
	entries = append(entries, e)
	return true, s.write("vocabulary.json", vocabulary{1, entries})
}
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	es, err := s.read()
	if err != nil {
		return err
	}
	out := make([]core.Entry, 0, len(es))
	for _, e := range es {
		if e.ID != id {
			out = append(out, e)
		}
	}
	return s.write("vocabulary.json", vocabulary{1, out})
}
func (s *Store) UpdateNote(id, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	es, err := s.read()
	if err != nil {
		return err
	}
	for i := range es {
		if es[i].ID == id {
			es[i].Note = note
			es[i].UpdatedAt = time.Now()
		}
	}
	return s.write("vocabulary.json", vocabulary{1, es})
}
func (s *Store) Export(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	es, e := s.read()
	if e != nil {
		return e
	}
	b, e := json.MarshalIndent(vocabulary{1, es}, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0600)
}
func (s *Store) write(name string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.Dir, ".save-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return replaceFile(tmp, filepath.Join(s.Dir, name))
}
