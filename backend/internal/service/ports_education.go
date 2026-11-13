package service

import "context"

// EducationRepository defines persistence operations for education.
type EducationRepository interface {
	ListSchools(context.Context) ([]ListSchoolsRecord, error)
	ListUserClasses(context.Context, string) ([]ListUserClassesRecord, error)
	CreateClass(context.Context, string, string, string) (CreateClassRecord, error)
	CanManageClass(context.Context, string, string, string) (CanManageClassRecord, error)
	AddClassMember(context.Context, string, string) (Command, error)
	ListClassStudents(context.Context, string) ([]ListClassStudentsRecord, error)
	GetClassStatistics(context.Context, string) (GetClassStatisticsRecord, error)
}
