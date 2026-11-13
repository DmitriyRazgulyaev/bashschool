package main

import "testing"

func TestSegmentSentences(t *testing.T) {
	s := "Беҙ бергә уҡыйбыҙ. Мин китап уҡыйым!"
	f := segmentLiterature(s, "sentence")
	if len(f) != 2 {
		t.Fatalf("want 2 sentences, got %d", len(f))
	}
	for _, item := range f {
		if s[item.Start:item.End] != item.Text {
			t.Fatalf("bad offsets: %#v", item)
		}
	}
}
func TestSegmentParagraphs(t *testing.T) {
	s := "Мин мәктәпкә барам. Һаумыһығыҙ!\n\nБөгөн көн матур. Мин уҡыйым."
	f := segmentLiterature(s, "paragraph")
	if len(f) != 2 {
		t.Fatalf("want 2 paragraphs got %d", len(f))
	}
	for _, item := range f {
		if s[item.Start:item.End] != item.Text {
			t.Fatalf("bad offsets: %#v", item)
		}
	}
}
