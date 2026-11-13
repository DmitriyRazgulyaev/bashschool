package main

import "github.com/gin-gonic/gin"

func schools(c *gin.Context) {
	rows, err := db.Query(c, `SELECT id,name FROM schools ORDER BY name`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка школ")
		return
	}
	defer rows.Close()
	arr := []gin.H{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			arr = append(arr, gin.H{"id": id, "name": name})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, arr)
}
func myClasses(c *gin.Context) {
	rows, err := db.Query(c, `SELECT cl.id,cl.name,s.name,cl.teacher_id FROM classes cl JOIN schools s ON s.id=cl.school_id WHERE cl.teacher_id=$1 OR EXISTS(SELECT 1 FROM class_members cm WHERE cm.class_id=cl.id AND cm.user_id=$1) ORDER BY cl.name`, uid(c))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка классов")
		return
	}
	defer rows.Close()
	arr := []gin.H{}
	for rows.Next() {
		var id, name, school, teacher string
		if err := rows.Scan(&id, &name, &school, &teacher); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			arr = append(arr, gin.H{"id": id, "name": name, "school": school, "teacherId": teacher})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, arr)
}
func createClass(c *gin.Context) {
	var in struct {
		SchoolID string `json:"schoolId" binding:"required,uuid"`
		Name     string `json:"name" binding:"required"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	var id string
	err := db.QueryRow(c, `INSERT INTO classes(school_id,teacher_id,name) VALUES($1,$2,$3) RETURNING id`, in.SchoolID, uid(c), in.Name).Scan(&id)
	if err != nil {
		fail(c, 422, "INVALID_CLASS", "Класс уже существует или школа не найдена")
		return
	}
	c.JSON(201, gin.H{"id": id, "name": in.Name})
}
func canManageClass(c *gin.Context) bool {
	var exists bool
	err := db.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM classes WHERE id=$1 AND (teacher_id=$2 OR $3='admin'))`, c.Param("id"), uid(c), c.GetString("role")).Scan(&exists)
	if err != nil || !exists {
		fail(c, 403, "FORBIDDEN", "Нет доступа к классу")
		return false
	}
	return true
}
func addStudent(c *gin.Context) {
	if !canManageClass(c) {
		return
	}
	var in struct {
		StudentID string `json:"studentId" binding:"required,uuid"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	tag, err := db.Exec(c, `INSERT INTO class_members(class_id,user_id) SELECT $1,id FROM users WHERE id=$2 AND role='student' ON CONFLICT DO UPDATE SET user_id=EXCLUDED.user_id`, c.Param("id"), in.StudentID)
	if err != nil || tag.RowsAffected() != 1 {
		fail(c, 422, "INVALID_STUDENT", "Не удалось добавить ученика")
		return
	}
	c.Status(204)
}
func classStudents(c *gin.Context) {
	if !canManageClass(c) {
		return
	}
	rows, err := db.Query(c, `SELECT u.id,u.display_name,u.xp,u.streak,(SELECT count(*) FROM tasks t WHERE t.user_id=u.id AND t.status='completed') FROM class_members cm JOIN users u ON u.id=cm.user_id WHERE cm.class_id=$1 ORDER BY u.xp DESC`, c.Param("id"))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка учеников")
		return
	}
	defer rows.Close()
	arr := []gin.H{}
	for rows.Next() {
		var id, name string
		var xp, streak, completed int
		if err := rows.Scan(&id, &name, &xp, &streak, &completed); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			arr = append(arr, gin.H{"userId": id, "displayName": name, "totalXp": xp, "currentStreak": streak, "completedTasks": completed})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, arr)
}
func classStatistics(c *gin.Context) {
	if !canManageClass(c) {
		return
	}
	var n, active, completed, xp int
	err := db.QueryRow(c, `SELECT count(*),count(*) FILTER(WHERE u.last_active_date>=CURRENT_DATE-7),COALESCE(sum((SELECT count(*) FROM tasks t WHERE t.user_id=u.id AND t.status='completed')),0),COALESCE(sum(u.xp),0) FROM class_members cm JOIN users u ON u.id=cm.user_id WHERE cm.class_id=$1`, c.Param("id")).Scan(&n, &active, &completed, &xp)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка отчёта")
		return
	}
	avg := 0.
	if n > 0 {
		avg = float64(xp) / float64(n)
	}
	jsonOK(c, gin.H{"classId": c.Param("id"), "studentsCount": n, "activeStudents": active, "totalCompletedTasks": completed, "averageXp": avg})
}
