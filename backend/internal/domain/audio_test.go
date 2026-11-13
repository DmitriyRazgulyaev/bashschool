package domain

import "testing"

func TestWebVTTCoordinates(t *testing.T) {
	body := "WEBVTT\n\n00:00.125 --> 00:02.500\nҺаумыһығыҙ!\n\n00:02.500 --> 00:04.000\nРәхмәт!\n"
	cues, err := ParseWebVTT(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].StartMS != 125 || cues[0].EndMS != 2500 {
		t.Fatalf("unexpected cues: %+v", cues)
	}
	for _, cue := range cues {
		if body[cue.StartOffset:cue.EndOffset] != cue.Text {
			t.Fatal("source coordinates must match original bytes")
		}
	}
}

func TestWebVTTRejectsInvalidTimelines(t *testing.T) {
	for _, body := range []string{"WEBVTT\n\n00:02.000 --> 00:01.000\nТекст", "WEBVTT\n\n00:00.000 --> 00:02.000\nПервый\n\n00:01.000 --> 00:03.000\nВторой", "not subtitles"} {
		if _, err := ParseWebVTT(body); err == nil {
			t.Errorf("expected invalid timeline: %q", body)
		}
	}
}
