package service

// Persistence results are application DTOs; no driver types cross this boundary.
type CreateAudioMaterialRecord struct {
	ID string
}

type GetAccessibleAudioRecord struct {
	Data        []byte
	ContentType string
	Name        string
}
