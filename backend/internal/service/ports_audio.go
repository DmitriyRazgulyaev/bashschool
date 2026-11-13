package service

import "context"

// AudioRepository defines persistence operations for audio.
type AudioRepository interface {
	CreateAudioMaterial(context.Context, string, string, string, string, string, []byte, string, string) (CreateAudioMaterialRecord, error)
	CreateAudioFragment(context.Context, string, string, string, string, int, int, int64, int64) (Command, error)
	GetAccessibleAudio(context.Context, string, string) (GetAccessibleAudioRecord, error)
}
