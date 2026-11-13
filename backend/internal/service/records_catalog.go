package service

type ListMaterialsRecord struct {
	ID      string
	Title   string
	Kind    string
	License string
	Body    string
	Created any
}

type GetPublishedFragmentRecord struct {
	ID      string
	Text    string
	Title   string
	Chapter string
	Start   *int
	End     *int
	StartMs *int64
	EndMs   *int64
}

type CreateMaterialRecord struct {
	ID string
}

type CreateFragmentRecord struct {
	ID string
}

type GetMaterialRecord struct {
	ID          string
	Title       string
	Kind        string
	License     string
	Description string
}

type ListPublishedFragmentsRecord struct {
	ID      string
	Text    string
	Title   string
	Chapter string
}
