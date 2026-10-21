package store

import (
	"github.com/simiooo/squirrel_translate/internal/core"
	"os"
	"path/filepath"
	"testing"
)

func sample() core.Entry {
	return core.Entry{SourceLanguage: "en", TargetLanguage: "简体中文", Definition: core.Dictionary{Word: "banks", Lemma: "bank", SourceLanguage: "en", Senses: []core.Sense{{PartOfSpeech: "noun", Meaning: "银行"}}}, Context: "I went to the bank.", Note: "财务"}
}
func TestVocabularySurvivesRestartAndDeduplicatesLemma(t *testing.T) {
	dir := t.TempDir()
	s, e := New(dir)
	if e != nil {
		t.Fatal(e)
	}
	if added, e := s.Save(sample()); e != nil || !added {
		t.Fatalf("first save: %v %v", added, e)
	}
	s2, _ := New(dir)
	entry := sample()
	entry.Definition.Word = "bank"
	entry.Note = ""
	entry.Definition.Senses[0].Meaning = "银行；河岸"
	if added, e := s2.Save(entry); e != nil || added {
		t.Fatalf("upsert: %v %v", added, e)
	}
	entries, e := s2.List()
	if e != nil || len(entries) != 1 {
		t.Fatalf("reload: %v %v", entries, e)
	}
	if entries[0].Note != "财务" || entries[0].Definition.Senses[0].Meaning != "银行；河岸" {
		t.Fatal("update lost fields")
	}
	if e = s2.Remove(entries[0].ID); e != nil {
		t.Fatal(e)
	}
	entries, _ = s2.List()
	if len(entries) != 0 {
		t.Fatal("remove did not persist")
	}
}
func TestCorruptVocabularyIsNeverOverwritten(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	p := filepath.Join(dir, "vocabulary.json")
	bad := []byte("{broken")
	_ = os.WriteFile(p, bad, 0600)
	if _, e := s.Save(sample()); e == nil {
		t.Fatal("accepted corrupt store")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(bad) {
		t.Fatal("destroyed original file")
	}
}
func TestIncompleteSenseCannotBeSaved(t *testing.T) {
	s, _ := New(t.TempDir())
	e := sample()
	e.Definition.Senses[0].Meaning = ""
	if _, err := s.Save(e); err == nil {
		t.Fatal("saved incomplete result")
	}
}
