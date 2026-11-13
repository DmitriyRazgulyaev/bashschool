package service

// Persistence results are application DTOs; no driver types cross this boundary.
type ListSchoolsRecord struct {
	ID   string
	Name string
}

type ListUserClassesRecord struct {
	ID      string
	Name    string
	School  string
	Teacher string
}

type CreateClassRecord struct {
	ID string
}

type CanManageClassRecord struct {
	Exists bool
}

type ListClassStudentsRecord struct {
	ID        string
	Name      string
	XP        int
	Streak    int
	Completed int
}

type GetClassStatisticsRecord struct {
	N         int
	Active    int
	Completed int
	XP        int
}
