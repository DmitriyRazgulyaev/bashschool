package service

import "context"

// EducationRegistrationRepository defines persistence operations for education registration.
type EducationRegistrationRepository interface {
	CreateClassStudent(context.Context, string, string, string, string, string) (CreateClassStudentRecord, error)
	EnrollCreatedStudent(context.Context, string, string) (Command, error)
	CreateSchool(context.Context, string) (CreateSchoolRecord, error)
}
