package service

type UpdateMeInput struct {
	DisplayName  *string `json:"displayName"`
	LanguagePair *string `json:"languagePair"`
	SchoolID     *string `json:"schoolId"`
	ClearSchool  bool    `json:"clearSchool"`
}
type SetRoleInput struct {
	Role string `json:"role" binding:"required,oneof=student teacher moderator admin"`
}
type UpdatePairInput struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}
type CreatePairInput struct {
	SourceLang string `json:"sourceLang" binding:"required"`
	TargetLang string `json:"targetLang" binding:"required"`
	Name       string `json:"name" binding:"required,min=3,max=120"`
}
type AddStudentInput struct {
	StudentID string `json:"studentId" binding:"required,uuid"`
}
type CreateClassInput struct {
	SchoolID string `json:"schoolId" binding:"required,uuid"`
	Name     string `json:"name" binding:"required"`
}
type CreateSchoolInput struct {
	Name string `json:"name" binding:"required,min=2,max=200"`
}
type AddFragmentInput struct {
	MaterialID  string `json:"materialId" binding:"required,uuid"`
	TextBa      string `json:"textBa" binding:"required"`
	SourceLang  string `json:"sourceLang"`
	Chapter     string `json:"chapter"`
	StartOffset *int   `json:"startOffset"`
	EndOffset   *int   `json:"endOffset"`
	StartTimeMs *int64 `json:"startTimeMs"`
	EndTimeMs   *int64 `json:"endTimeMs"`
}
type AddMaterialInput struct {
	Title       string `json:"title" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=text audio"`
	License     string `json:"license" binding:"required"`
	Description string `json:"description"`
	SourceURL   string `json:"sourceUrl"`
}
type AddCandidateInput struct {
	Translation  string `json:"translation" binding:"required,min=1,max=2000"`
	LanguagePair string `json:"languagePair"`
}
type ResolveCaseInput struct {
	CandidateID string `json:"candidateId" binding:"required,uuid"`
	Comment     string `json:"comment"`
}
type MergeCandidatesInput struct {
	CandidateID string `json:"candidateId" binding:"required,uuid"`
	CanonicalID string `json:"canonicalId" binding:"required,uuid"`
}
type SaveManualTranslationsInput struct {
	LanguagePair string `json:"languagePair" binding:"required"`
	Correct      string `json:"correct" binding:"required,max=2000"`
	Incorrect    string `json:"incorrect" binding:"required,max=2000"`
}
type PublishLiteratureFragmentInput struct {
	LanguagePair string `json:"languagePair" binding:"required"`
}

type UploadFields map[string]string

func (f UploadFields) Field(key string) string { return f[key] }
func (f UploadFields) FieldDefault(key, fallback string) string {
	if value, ok := f[key]; ok {
		return value
	}
	return fallback
}

type LiteratureUpload struct {
	UploadFields
	Text      string
	ReadError error
}
type AudioUpload struct {
	ContentType string
	UploadFields
	Audio     []byte
	Filename  string
	ReadError error
}
