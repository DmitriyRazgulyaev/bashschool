package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Schools(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListSchools(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка школ")
	}

	arr := []domain.Object{}
	for _, row := range rows1 {
		var id, name string
		{
			id, name = row.ID, row.Name

			arr = append(arr, domain.Object{"id": id, "name": name})
		}

	}

	return domain.OK(arr), nil
}
func (s *Service) MyClasses(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListUserClasses(ctx, p.Actor.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка классов")
	}

	arr := []domain.Object{}
	for _, row := range rows1 {
		var id, name, school, teacher string
		{
			id, name, school, teacher = row.ID, row.Name, row.School, row.Teacher

			arr = append(arr, domain.Object{
				"id":        id,
				"name":      name,
				"school":    school,
				"teacherId": teacher,
			})
		}

	}

	return domain.OK(arr), nil
}
func (s *Service) CreateClass(ctx context.Context, p domain.Parameters, in CreateClassInput) (domain.Result, error) {
	var id string
	recordCreateClass, err := s.store.CreateClass(ctx, in.SchoolID, p.Actor.ID, in.Name)
	id = recordCreateClass.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_CLASS", "Класс уже существует или школа не найдена")
	}
	return domain.New(domain.Object{"id": id, "name": in.Name}), nil
}
func (s *Service) canManageClass(ctx context.Context, p domain.Parameters) error {
	var exists bool
	recordCanManageClass, err := s.store.CanManageClass(ctx, p.ID, p.Actor.ID, p.Actor.Role)
	exists = recordCanManageClass.Exists

	if err != nil {
		return err
	}
	if !exists {
		return domain.Failure(domain.Forbidden, "FORBIDDEN", "Нет доступа к классу")
	}
	return nil
}
func (s *Service) AddStudent(ctx context.Context, p domain.Parameters, in AddStudentInput) (domain.Result, error) {
	if err := s.canManageClass(ctx, p); err != nil {
		return domain.Result{}, err
	}
	tag, err := s.store.AddClassMember(ctx, p.ID, in.StudentID)
	if err != nil || tag.RowsAffected() != 1 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_STUDENT", "Не удалось добавить ученика")
	}
	return domain.Empty(), nil
}
func (s *Service) ClassStudents(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	if err := s.canManageClass(ctx, p); err != nil {
		return domain.Result{}, err
	}
	rows1, err := s.store.ListClassStudents(ctx, p.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка учеников")
	}

	arr := []domain.Object{}
	for _, row := range rows1 {
		var id, name string
		var xp, streak, completed int
		{
			id, name, xp, streak, completed = row.ID, row.Name, row.XP, row.Streak,
				row.Completed

			arr = append(arr, domain.Object{
				"userId":         id,
				"displayName":    name,
				"totalXp":        xp,
				"currentStreak":  streak,
				"completedTasks": completed,
			})
		}

	}

	return domain.OK(arr), nil
}
func (s *Service) ClassStatistics(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	if err := s.canManageClass(ctx, p); err != nil {
		return domain.Result{}, err
	}
	var n, active, completed, xp int
	recordGetClassStatistics, err := s.store.GetClassStatistics(ctx, p.ID)
	n, active, completed, xp = recordGetClassStatistics.N, recordGetClassStatistics.Active, recordGetClassStatistics.Completed, recordGetClassStatistics.XP

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка отчёта")
	}
	avg := 0.0
	if n > 0 {
		avg = float64(xp) / float64(n)
	}
	return domain.OK(domain.Object{
		"classId":             p.ID,
		"studentsCount":       n,
		"activeStudents":      active,
		"totalCompletedTasks": completed,
		"averageXp":           avg,
	}), nil
}
