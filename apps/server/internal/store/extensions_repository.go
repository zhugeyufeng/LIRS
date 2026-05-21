package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func optionalID(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func optionalText(value string) string {
	return strings.TrimSpace(value)
}

func (r *Repository) TrainingCourses(ctx context.Context) ([]TrainingCourse, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT tc.id::text, tc.tenant_id::text, tc.title, tc.category, COALESCE(tc.instrument_id::text, ''), COALESCE(i.name, ''),
       tc.instructor, tc.delivery_mode, tc.duration_hours::float8, tc.required_for_booking, tc.status, tc.description, tc.created_at, tc.updated_at
FROM training_courses tc
LEFT JOIN instruments i ON i.id = tc.instrument_id
WHERE ($1::boolean OR tc.tenant_id = $2::uuid)
ORDER BY tc.created_at DESC, tc.title
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingCourse, 0)
	for rows.Next() {
		var item TrainingCourse
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Title, &item.Category, &item.InstrumentID, &item.InstrumentName, &item.Instructor, &item.DeliveryMode, &item.DurationHours, &item.RequiredForBooking, &item.Status, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingCourse(ctx context.Context, id string, input TrainingCourseInput) (TrainingCourse, error) {
	tenant := TenantFromContext(ctx)
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Instructor = strings.TrimSpace(input.Instructor)
	input.DeliveryMode = strings.TrimSpace(strings.ToLower(input.DeliveryMode))
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Title == "" {
		return TrainingCourse{}, clientError("training course title is required")
	}
	if input.Category == "" {
		input.Category = "仪器培训"
	}
	if input.DeliveryMode == "" {
		input.DeliveryMode = "blended"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.DurationHours < 0 {
		input.DurationHours = 0
	}
	var item TrainingCourse
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_courses (tenant_id, title, category, instrument_id, instructor, delivery_mode, duration_hours, required_for_booking, status, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id::text, tenant_id::text, title, category, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), instructor, delivery_mode, duration_hours::float8, required_for_booking, status, description, created_at, updated_at
`, tenant.TenantID, input.Title, input.Category, optionalID(input.InstrumentID), input.Instructor, input.DeliveryMode, input.DurationHours, input.RequiredForBooking, input.Status, input.Description).Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Category, &item.InstrumentID, &item.InstrumentName, &item.Instructor, &item.DeliveryMode, &item.DurationHours, &item.RequiredForBooking, &item.Status, &item.Description, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.course.create", "training_course", item.ID, "", item.Title)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_courses
SET title = $2,
    category = $3,
    instrument_id = $4,
    instructor = $5,
    delivery_mode = $6,
    duration_hours = $7,
    required_for_booking = $8,
    status = $9,
    description = $10,
    updated_at = now()
WHERE id = $1 AND ($11::boolean OR tenant_id = $12::uuid)
RETURNING id::text, tenant_id::text, title, category, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), instructor, delivery_mode, duration_hours::float8, required_for_booking, status, description, created_at, updated_at
`, id, input.Title, input.Category, optionalID(input.InstrumentID), input.Instructor, input.DeliveryMode, input.DurationHours, input.RequiredForBooking, input.Status, input.Description, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.Category, &item.InstrumentID, &item.InstrumentName, &item.Instructor, &item.DeliveryMode, &item.DurationHours, &item.RequiredForBooking, &item.Status, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.course.update", "training_course", item.ID, "", item.Title)
	}
	return item, err
}

func (r *Repository) TrainingAuthorizations(ctx context.Context) ([]TrainingAuthorization, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT ta.id::text, ta.tenant_id::text, COALESCE(ta.user_id::text, ''), ta.user_name, COALESCE(ta.course_id::text, ''), COALESCE(tc.title, ''), COALESCE(ta.instrument_id::text, ''), COALESCE(i.name, ''),
       ta.status, ta.expires_at, ta.notes, ta.created_at, ta.updated_at
FROM training_authorizations ta
LEFT JOIN training_courses tc ON tc.id = ta.course_id
LEFT JOIN instruments i ON i.id = ta.instrument_id
WHERE ($1::boolean OR ta.tenant_id = $2::uuid)
ORDER BY ta.created_at DESC, ta.expires_at DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingAuthorization, 0)
	for rows.Next() {
		var item TrainingAuthorization
		if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.InstrumentID, &item.InstrumentName, &item.Status, &item.ExpiresAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingAuthorization(ctx context.Context, id string, input TrainingAuthorizationInput) (TrainingAuthorization, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.CourseID = strings.TrimSpace(input.CourseID)
	input.InstrumentID = strings.TrimSpace(input.InstrumentID)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.UserName == "" {
		return TrainingAuthorization{}, clientError("user name is required")
	}
	if input.Status == "" {
		input.Status = "pending"
	}
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().Add(180 * 24 * time.Hour)
	}
	var item TrainingAuthorization
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_authorizations (tenant_id, user_id, user_name, course_id, instrument_id, status, expires_at, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(course_id::text, ''), COALESCE((SELECT title FROM training_courses WHERE id = course_id), ''), COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), status, expires_at, notes, created_at, updated_at
`, tenant.TenantID, optionalID(input.UserID), input.UserName, optionalID(input.CourseID), optionalID(input.InstrumentID), input.Status, input.ExpiresAt, input.Notes).Scan(
			&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.InstrumentID, &item.InstrumentName, &item.Status, &item.ExpiresAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.authorization.create", "training_authorization", item.ID, "", item.UserName)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_authorizations
SET user_id = $2,
    user_name = $3,
    course_id = $4,
    instrument_id = $5,
    status = $6,
    expires_at = $7,
    notes = $8,
    updated_at = now()
WHERE id = $1 AND ($9::boolean OR tenant_id = $10::uuid)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(course_id::text, ''), COALESCE((SELECT title FROM training_courses WHERE id = course_id), ''), COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), status, expires_at, notes, created_at, updated_at
`, id, optionalID(input.UserID), input.UserName, optionalID(input.CourseID), optionalID(input.InstrumentID), input.Status, input.ExpiresAt, input.Notes, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.InstrumentID, &item.InstrumentName, &item.Status, &item.ExpiresAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.authorization.update", "training_authorization", item.ID, "", item.UserName)
	}
	return item, err
}

func (r *Repository) TrainingQuestions(ctx context.Context) ([]TrainingQuestion, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, title, question_type, options, correct_answer, explanation, status, created_at, updated_at
FROM training_questions
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY created_at DESC, title
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingQuestion, 0)
	for rows.Next() {
		var item TrainingQuestion
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Title, &item.QuestionType, &item.Options, &item.CorrectAnswer, &item.Explanation, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingQuestion(ctx context.Context, id string, input TrainingQuestionInput) (TrainingQuestion, error) {
	tenant := TenantFromContext(ctx)
	input.Title = strings.TrimSpace(input.Title)
	input.QuestionType = strings.TrimSpace(strings.ToLower(input.QuestionType))
	input.Options = strings.TrimSpace(input.Options)
	input.CorrectAnswer = strings.TrimSpace(input.CorrectAnswer)
	input.Explanation = strings.TrimSpace(input.Explanation)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Title == "" {
		return TrainingQuestion{}, clientError("question title is required")
	}
	if input.QuestionType == "" {
		input.QuestionType = "single"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	var item TrainingQuestion
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_questions (tenant_id, title, question_type, options, correct_answer, explanation, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id::text, tenant_id::text, title, question_type, options, correct_answer, explanation, status, created_at, updated_at
`, tenant.TenantID, input.Title, input.QuestionType, input.Options, input.CorrectAnswer, input.Explanation, input.Status).Scan(
			&item.ID, &item.TenantID, &item.Title, &item.QuestionType, &item.Options, &item.CorrectAnswer, &item.Explanation, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.question.create", "training_question", item.ID, "", item.Title)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_questions
SET title = $2,
    question_type = $3,
    options = $4,
    correct_answer = $5,
    explanation = $6,
    status = $7,
    updated_at = now()
WHERE id = $1 AND ($8::boolean OR tenant_id = $9::uuid)
RETURNING id::text, tenant_id::text, title, question_type, options, correct_answer, explanation, status, created_at, updated_at
`, id, input.Title, input.QuestionType, input.Options, input.CorrectAnswer, input.Explanation, input.Status, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.QuestionType, &item.Options, &item.CorrectAnswer, &item.Explanation, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.question.update", "training_question", item.ID, "", item.Title)
	}
	return item, err
}

func (r *Repository) TrainingExams(ctx context.Context) ([]TrainingExam, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT te.id::text, te.tenant_id::text, COALESCE(te.user_id::text, ''), te.user_name, COALESCE(te.course_id::text, ''), COALESCE(tc.title, ''), te.score::float8, te.passed, te.answers, te.status, te.notes, te.exam_at, te.created_at, te.updated_at
FROM training_exams te
LEFT JOIN training_courses tc ON tc.id = te.course_id
WHERE ($1::boolean OR te.tenant_id = $2::uuid)
ORDER BY te.exam_at DESC, te.created_at DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingExam, 0)
	for rows.Next() {
		var item TrainingExam
		if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.Score, &item.Passed, &item.Answers, &item.Status, &item.Notes, &item.ExamAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingExam(ctx context.Context, id string, input TrainingExamInput) (TrainingExam, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.CourseID = strings.TrimSpace(input.CourseID)
	input.Answers = strings.TrimSpace(input.Answers)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.UserName == "" {
		input.UserName = input.Actor
	}
	if input.Status == "" {
		input.Status = "submitted"
	}
	if input.Score < 0 {
		input.Score = 0
	}
	if input.Score > 100 {
		input.Score = 100
	}
	if input.ExamAt.IsZero() {
		input.ExamAt = time.Now()
	}
	passed := input.Passed
	if input.Status == "graded" && input.Score >= 60 {
		passed = true
	}
	var item TrainingExam
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_exams (tenant_id, user_id, user_name, course_id, score, passed, answers, status, notes, exam_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(course_id::text, ''), COALESCE((SELECT title FROM training_courses WHERE id = course_id), ''), score::float8, passed, answers, status, notes, exam_at, created_at, updated_at
`, tenant.TenantID, optionalID(input.UserID), input.UserName, optionalID(input.CourseID), input.Score, passed, input.Answers, input.Status, input.Notes, input.ExamAt).Scan(
			&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.Score, &item.Passed, &item.Answers, &item.Status, &item.Notes, &item.ExamAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.exam.create", "training_exam", item.ID, "", item.UserName)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_exams
SET user_id = $2,
    user_name = $3,
    course_id = $4,
    score = $5,
    passed = $6,
    answers = $7,
    status = $8,
    notes = $9,
    exam_at = $10,
    updated_at = now()
WHERE id = $1 AND ($11::boolean OR tenant_id = $12::uuid)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(course_id::text, ''), COALESCE((SELECT title FROM training_courses WHERE id = course_id), ''), score::float8, passed, answers, status, notes, exam_at, created_at, updated_at
`, id, optionalID(input.UserID), input.UserName, optionalID(input.CourseID), input.Score, passed, input.Answers, input.Status, input.Notes, input.ExamAt, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.CourseID, &item.CourseTitle, &item.Score, &item.Passed, &item.Answers, &item.Status, &item.Notes, &item.ExamAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.exam.update", "training_exam", item.ID, "", item.UserName)
	}
	return item, err
}

func (r *Repository) TrainingPracticals(ctx context.Context) ([]TrainingPractical, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT tp.id::text, tp.tenant_id::text, COALESCE(tp.user_id::text, ''), tp.user_name, COALESCE(tp.instrument_id::text, ''), COALESCE(i.name, ''), tp.assessor, tp.score::float8, tp.result, tp.notes, tp.assessment_at, tp.created_at, tp.updated_at
FROM training_practical_assessments tp
LEFT JOIN instruments i ON i.id = tp.instrument_id
WHERE ($1::boolean OR tp.tenant_id = $2::uuid)
ORDER BY tp.assessment_at DESC, tp.created_at DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingPractical, 0)
	for rows.Next() {
		var item TrainingPractical
		if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.InstrumentID, &item.InstrumentName, &item.Assessor, &item.Score, &item.Result, &item.Notes, &item.AssessmentAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingPractical(ctx context.Context, id string, input TrainingPracticalInput) (TrainingPractical, error) {
	tenant := TenantFromContext(ctx)
	input.UserID = strings.TrimSpace(input.UserID)
	input.UserName = strings.TrimSpace(input.UserName)
	input.InstrumentID = strings.TrimSpace(input.InstrumentID)
	input.Assessor = strings.TrimSpace(input.Assessor)
	input.Result = strings.TrimSpace(strings.ToLower(input.Result))
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.UserName == "" {
		input.UserName = input.Actor
	}
	if input.Assessor == "" {
		input.Assessor = input.Actor
	}
	if input.Result == "" {
		input.Result = "pending"
	}
	if input.Score < 0 {
		input.Score = 0
	}
	if input.Score > 100 {
		input.Score = 100
	}
	if input.AssessmentAt.IsZero() {
		input.AssessmentAt = time.Now()
	}
	var item TrainingPractical
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_practical_assessments (tenant_id, user_id, user_name, instrument_id, assessor, score, result, notes, assessment_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), assessor, score::float8, result, notes, assessment_at, created_at, updated_at
`, tenant.TenantID, optionalID(input.UserID), input.UserName, optionalID(input.InstrumentID), input.Assessor, input.Score, input.Result, input.Notes, input.AssessmentAt).Scan(
			&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.InstrumentID, &item.InstrumentName, &item.Assessor, &item.Score, &item.Result, &item.Notes, &item.AssessmentAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.practical.create", "training_practical_assessment", item.ID, "", item.UserName)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_practical_assessments
SET user_id = $2,
    user_name = $3,
    instrument_id = $4,
    assessor = $5,
    score = $6,
    result = $7,
    notes = $8,
    assessment_at = $9,
    updated_at = now()
WHERE id = $1 AND ($10::boolean OR tenant_id = $11::uuid)
RETURNING id::text, tenant_id::text, COALESCE(user_id::text, ''), user_name, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), assessor, score::float8, result, notes, assessment_at, created_at, updated_at
`, id, optionalID(input.UserID), input.UserName, optionalID(input.InstrumentID), input.Assessor, input.Score, input.Result, input.Notes, input.AssessmentAt, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.UserID, &item.UserName, &item.InstrumentID, &item.InstrumentName, &item.Assessor, &item.Score, &item.Result, &item.Notes, &item.AssessmentAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.practical.update", "training_practical_assessment", item.ID, "", item.UserName)
	}
	return item, err
}

func (r *Repository) TrainingRules(ctx context.Context) ([]TrainingRule, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT tr.id::text, tr.tenant_id::text, COALESCE(tr.instrument_id::text, ''), COALESCE(i.name, ''), tr.require_training, tr.require_exam, tr.require_approval, tr.min_score::float8, tr.status, tr.notes, tr.created_at, tr.updated_at
FROM training_rules tr
LEFT JOIN instruments i ON i.id = tr.instrument_id
WHERE ($1::boolean OR tr.tenant_id = $2::uuid)
ORDER BY tr.updated_at DESC, i.name
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrainingRule, 0)
	for rows.Next() {
		var item TrainingRule
		if err := rows.Scan(&item.ID, &item.TenantID, &item.InstrumentID, &item.InstrumentName, &item.RequireTraining, &item.RequireExam, &item.RequireApproval, &item.MinScore, &item.Status, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveTrainingRule(ctx context.Context, id string, input TrainingRuleInput) (TrainingRule, error) {
	tenant := TenantFromContext(ctx)
	input.InstrumentID = strings.TrimSpace(input.InstrumentID)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.InstrumentID == "" {
		return TrainingRule{}, clientError("instrument is required")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.MinScore < 0 {
		input.MinScore = 0
	}
	if input.MinScore > 100 {
		input.MinScore = 100
	}
	var item TrainingRule
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO training_rules (tenant_id, instrument_id, require_training, require_exam, require_approval, min_score, status, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id::text, tenant_id::text, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), require_training, require_exam, require_approval, min_score::float8, status, notes, created_at, updated_at
`, tenant.TenantID, optionalID(input.InstrumentID), input.RequireTraining, input.RequireExam, input.RequireApproval, input.MinScore, input.Status, input.Notes).Scan(
			&item.ID, &item.TenantID, &item.InstrumentID, &item.InstrumentName, &item.RequireTraining, &item.RequireExam, &item.RequireApproval, &item.MinScore, &item.Status, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "training.rule.create", "training_rule", item.ID, "", item.InstrumentName)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE training_rules
SET instrument_id = $2,
    require_training = $3,
    require_exam = $4,
    require_approval = $5,
    min_score = $6,
    status = $7,
    notes = $8,
    updated_at = now()
WHERE id = $1 AND ($9::boolean OR tenant_id = $10::uuid)
RETURNING id::text, tenant_id::text, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), require_training, require_exam, require_approval, min_score::float8, status, notes, created_at, updated_at
`, id, optionalID(input.InstrumentID), input.RequireTraining, input.RequireExam, input.RequireApproval, input.MinScore, input.Status, input.Notes, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.InstrumentID, &item.InstrumentName, &item.RequireTraining, &item.RequireExam, &item.RequireApproval, &item.MinScore, &item.Status, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "training.rule.update", "training_rule", item.ID, "", item.InstrumentName)
	}
	return item, err
}

func (r *Repository) BusinessConfigs(ctx context.Context, domain string, kind string) ([]BusinessConfig, error) {
	tenant := TenantFromContext(ctx)
	domain, kind, err := normalizeBusinessConfigRoute(domain, kind)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, domain, kind, title, category, scope, status, description, config_json::text, updated_by, created_at, updated_at
FROM business_configs
WHERE domain = $1
  AND kind = $2
  AND ($3::boolean OR tenant_id = $4::uuid)
ORDER BY updated_at DESC, title
`, domain, kind, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BusinessConfig, 0)
	for rows.Next() {
		var item BusinessConfig
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Domain, &item.Kind, &item.Title, &item.Category, &item.Scope, &item.Status, &item.Description, &item.ConfigJSON, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveBusinessConfig(ctx context.Context, domain string, kind string, id string, input BusinessConfigInput) (BusinessConfig, error) {
	tenant := TenantFromContext(ctx)
	domain, kind, err := normalizeBusinessConfigRoute(domain, kind)
	if err != nil {
		return BusinessConfig{}, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Title == "" {
		return BusinessConfig{}, clientError("config title is required")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if !validBusinessConfigStatus(input.Status) {
		return BusinessConfig{}, clientError("invalid config status")
	}
	configJSON, err := normalizeConfigJSON(input.ConfigJSON)
	if err != nil {
		return BusinessConfig{}, err
	}

	var item BusinessConfig
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO business_configs (tenant_id, domain, kind, title, category, scope, status, description, config_json, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
RETURNING id::text, tenant_id::text, domain, kind, title, category, scope, status, description, config_json::text, updated_by, created_at, updated_at
`, tenant.TenantID, domain, kind, input.Title, input.Category, input.Scope, input.Status, input.Description, configJSON, input.Actor).Scan(
			&item.ID, &item.TenantID, &item.Domain, &item.Kind, &item.Title, &item.Category, &item.Scope, &item.Status, &item.Description, &item.ConfigJSON, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, domain+".config.create", "business_config", item.ID, "", item.Title)
		}
		return item, err
	}

	err = r.db.QueryRow(ctx, `
UPDATE business_configs
SET title = $2,
    category = $3,
    scope = $4,
    status = $5,
    description = $6,
    config_json = $7::jsonb,
    updated_by = $8,
    updated_at = now()
WHERE id = $1
  AND domain = $9
  AND kind = $10
  AND ($11::boolean OR tenant_id = $12::uuid)
RETURNING id::text, tenant_id::text, domain, kind, title, category, scope, status, description, config_json::text, updated_by, created_at, updated_at
`, id, input.Title, input.Category, input.Scope, input.Status, input.Description, configJSON, input.Actor, domain, kind, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Domain, &item.Kind, &item.Title, &item.Category, &item.Scope, &item.Status, &item.Description, &item.ConfigJSON, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, domain+".config.update", "business_config", item.ID, "", item.Title)
	}
	return item, err
}

func normalizeBusinessConfigRoute(domain string, kind string) (string, string, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	kind = strings.TrimSpace(strings.ToLower(kind))
	allowed := map[string]map[string]bool{
		"workflow": {
			"templates":  true,
			"rules":      true,
			"approvers":  true,
			"instances":  true,
			"exceptions": true,
		},
		"billing": {
			"instrument-rules": true,
			"material-rules":   true,
			"invoices":         true,
		},
	}
	if allowed[domain][kind] {
		return domain, kind, nil
	}
	return "", "", clientError("invalid config kind")
}

func validBusinessConfigStatus(status string) bool {
	switch status {
	case "draft", "active", "disabled", "archived":
		return true
	default:
		return false
	}
}

func normalizeConfigJSON(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "{}", nil
	}
	if !json.Valid([]byte(value)) {
		return "", clientError("invalid config json")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(value)); err != nil {
		return "", clientError("invalid config json")
	}
	return compact.String(), nil
}

func (r *Repository) Spaces(ctx context.Context) ([]Space, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, name, kind, department, location, capacity, status, access_control_point, description, created_at, updated_at
FROM spaces
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY kind, name
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Space, 0)
	for rows.Next() {
		var item Space
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.Kind, &item.Department, &item.Location, &item.Capacity, &item.Status, &item.AccessControlPoint, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveSpace(ctx context.Context, id string, input SpaceInput) (Space, error) {
	tenant := TenantFromContext(ctx)
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = strings.TrimSpace(strings.ToLower(input.Kind))
	input.Department = strings.TrimSpace(input.Department)
	input.Location = strings.TrimSpace(input.Location)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.AccessControlPoint = strings.TrimSpace(input.AccessControlPoint)
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Name == "" {
		return Space{}, clientError("space name is required")
	}
	if input.Kind == "" {
		input.Kind = "lab"
	}
	if input.Status == "" {
		input.Status = "available"
	}
	if input.Capacity <= 0 {
		input.Capacity = 1
	}
	var item Space
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO spaces (tenant_id, name, kind, department, location, capacity, status, access_control_point, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id::text, tenant_id::text, name, kind, department, location, capacity, status, access_control_point, description, created_at, updated_at
`, tenant.TenantID, input.Name, input.Kind, input.Department, input.Location, input.Capacity, input.Status, input.AccessControlPoint, input.Description).Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Kind, &item.Department, &item.Location, &item.Capacity, &item.Status, &item.AccessControlPoint, &item.Description, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "space.create", "space", item.ID, "", item.Name)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE spaces
SET name = $2,
    kind = $3,
    department = $4,
    location = $5,
    capacity = $6,
    status = $7,
    access_control_point = $8,
    description = $9,
    updated_at = now()
WHERE id = $1 AND ($10::boolean OR tenant_id = $11::uuid)
RETURNING id::text, tenant_id::text, name, kind, department, location, capacity, status, access_control_point, description, created_at, updated_at
`, id, input.Name, input.Kind, input.Department, input.Location, input.Capacity, input.Status, input.AccessControlPoint, input.Description, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Kind, &item.Department, &item.Location, &item.Capacity, &item.Status, &item.AccessControlPoint, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "space.update", "space", item.ID, "", item.Name)
	}
	return item, err
}

func (r *Repository) SpaceReservations(ctx context.Context) ([]SpaceReservation, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT sr.id::text, sr.tenant_id::text, sr.space_id::text, sp.name, COALESCE(sr.requester_id::text, ''), sr.requester, sr.purpose, lower(sr.period), upper(sr.period), sr.status, sr.created_at
FROM space_reservations sr
JOIN spaces sp ON sp.id = sr.space_id
WHERE ($1::boolean OR sr.tenant_id = $2::uuid)
ORDER BY sr.created_at DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SpaceReservation, 0)
	for rows.Next() {
		var item SpaceReservation
		if err := rows.Scan(&item.ID, &item.TenantID, &item.SpaceID, &item.SpaceName, &item.RequesterID, &item.Requester, &item.Purpose, &item.StartTime, &item.EndTime, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSpaceReservation(ctx context.Context, input SpaceReservationInput) (SpaceReservation, error) {
	tenant := TenantFromContext(ctx)
	input.SpaceID = strings.TrimSpace(input.SpaceID)
	input.Requester = strings.TrimSpace(input.Requester)
	input.Purpose = strings.TrimSpace(input.Purpose)
	input.Actor = strings.TrimSpace(input.Actor)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.SpaceID == "" || input.Requester == "" || input.Purpose == "" {
		return SpaceReservation{}, clientError("space reservation input is incomplete")
	}
	if !input.EndTime.After(input.StartTime) {
		return SpaceReservation{}, clientError("reservation end time must be after start time")
	}
	var spaceStatus string
	if err := r.db.QueryRow(ctx, `
SELECT status
FROM spaces
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
`, input.SpaceID, tenant.AllTenants, tenant.TenantID).Scan(&spaceStatus); err != nil {
		return SpaceReservation{}, err
	}
	if spaceStatus == "disabled" || spaceStatus == "maintenance" {
		return SpaceReservation{}, clientError("space is unavailable")
	}
	var conflict bool
	if err := r.db.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM space_reservations
    WHERE space_id = $1
      AND status IN ('pending', 'approved', 'in_use')
      AND period && tstzrange($2, $3, '[)')
      AND ($4::boolean OR tenant_id = $5::uuid)
)
`, input.SpaceID, input.StartTime, input.EndTime, tenant.AllTenants, tenant.TenantID).Scan(&conflict); err != nil {
		return SpaceReservation{}, err
	}
	if conflict {
		return SpaceReservation{}, clientError("space reservation conflicts with an existing booking")
	}
	var item SpaceReservation
	err := r.db.QueryRow(ctx, `
INSERT INTO space_reservations (tenant_id, space_id, requester_id, requester, purpose, period, status)
VALUES ($1, $2, $3, $4, $5, tstzrange($6, $7, '[)'), 'pending')
RETURNING id::text, tenant_id::text, space_id::text, (SELECT name FROM spaces WHERE id = space_id), COALESCE(requester_id::text, ''), requester, purpose, lower(period), upper(period), status, created_at
`, tenant.TenantID, input.SpaceID, optionalID(input.RequesterID), input.Requester, input.Purpose, input.StartTime, input.EndTime).Scan(
		&item.ID, &item.TenantID, &item.SpaceID, &item.SpaceName, &item.RequesterID, &item.Requester, &item.Purpose, &item.StartTime, &item.EndTime, &item.Status, &item.CreatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "space_reservation.create", "space_reservation", item.ID, "", item.Purpose)
	}
	return item, err
}

func (r *Repository) Samples(ctx context.Context) ([]Sample, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, code, name, COALESCE(owner_id::text, ''), owner_name, department, group_name, location, status, hazard_level, storage_condition, description, created_at, updated_at
FROM samples
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY created_at DESC, code
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Sample, 0)
	for rows.Next() {
		var item Sample
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Code, &item.Name, &item.OwnerID, &item.OwnerName, &item.Department, &item.GroupName, &item.Location, &item.Status, &item.HazardLevel, &item.StorageCondition, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveSample(ctx context.Context, id string, input SampleInput) (Sample, error) {
	tenant := TenantFromContext(ctx)
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.OwnerName = strings.TrimSpace(input.OwnerName)
	input.Department = strings.TrimSpace(input.Department)
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.Location = strings.TrimSpace(input.Location)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.HazardLevel = strings.TrimSpace(strings.ToLower(input.HazardLevel))
	input.StorageCondition = strings.TrimSpace(input.StorageCondition)
	input.Description = strings.TrimSpace(input.Description)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Code == "" || input.Name == "" {
		return Sample{}, clientError("sample code and name are required")
	}
	if input.Status == "" {
		input.Status = "stored"
	}
	if input.HazardLevel == "" {
		input.HazardLevel = "normal"
	}
	if input.OwnerName == "" {
		input.OwnerName = input.Actor
	}
	var item Sample
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO samples (tenant_id, code, name, owner_id, owner_name, department, group_name, location, status, hazard_level, storage_condition, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id::text, tenant_id::text, code, name, COALESCE(owner_id::text, ''), owner_name, department, group_name, location, status, hazard_level, storage_condition, description, created_at, updated_at
`, tenant.TenantID, input.Code, input.Name, optionalID(input.OwnerID), input.OwnerName, input.Department, input.GroupName, input.Location, input.Status, input.HazardLevel, input.StorageCondition, input.Description).Scan(
			&item.ID, &item.TenantID, &item.Code, &item.Name, &item.OwnerID, &item.OwnerName, &item.Department, &item.GroupName, &item.Location, &item.Status, &item.HazardLevel, &item.StorageCondition, &item.Description, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "sample.create", "sample", item.ID, "", item.Code)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE samples
SET code = $2,
    name = $3,
    owner_id = $4,
    owner_name = $5,
    department = $6,
    group_name = $7,
    location = $8,
    status = $9,
    hazard_level = $10,
    storage_condition = $11,
    description = $12,
    updated_at = now()
WHERE id = $1 AND ($13::boolean OR tenant_id = $14::uuid)
RETURNING id::text, tenant_id::text, code, name, COALESCE(owner_id::text, ''), owner_name, department, group_name, location, status, hazard_level, storage_condition, description, created_at, updated_at
`, id, input.Code, input.Name, optionalID(input.OwnerID), input.OwnerName, input.Department, input.GroupName, input.Location, input.Status, input.HazardLevel, input.StorageCondition, input.Description, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Code, &item.Name, &item.OwnerID, &item.OwnerName, &item.Department, &item.GroupName, &item.Location, &item.Status, &item.HazardLevel, &item.StorageCondition, &item.Description, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "sample.update", "sample", item.ID, "", item.Code)
	}
	return item, err
}

func (r *Repository) SampleMovements(ctx context.Context) ([]SampleMovement, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT sm.id::text, sm.tenant_id::text, sm.sample_id::text, s.code, s.name, sm.movement_type, sm.from_location, sm.to_location, sm.reason, sm.created_at
FROM sample_movements sm
JOIN samples s ON s.id = sm.sample_id
WHERE ($1::boolean OR sm.tenant_id = $2::uuid)
ORDER BY sm.created_at DESC
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SampleMovement, 0)
	for rows.Next() {
		var item SampleMovement
		if err := rows.Scan(&item.ID, &item.TenantID, &item.SampleID, &item.SampleCode, &item.SampleName, &item.MovementType, &item.FromLocation, &item.ToLocation, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSampleMovement(ctx context.Context, input SampleMovementInput) (SampleMovement, error) {
	tenant := TenantFromContext(ctx)
	input.SampleID = strings.TrimSpace(input.SampleID)
	input.MovementType = strings.TrimSpace(strings.ToLower(input.MovementType))
	input.FromLocation = strings.TrimSpace(input.FromLocation)
	input.ToLocation = strings.TrimSpace(input.ToLocation)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.SampleID == "" || input.MovementType == "" {
		return SampleMovement{}, clientError("sample movement input is incomplete")
	}
	var item SampleMovement
	err := r.db.QueryRow(ctx, `
INSERT INTO sample_movements (tenant_id, sample_id, movement_type, from_location, to_location, reason)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id::text, tenant_id::text, sample_id::text, (SELECT code FROM samples WHERE id = sample_id), (SELECT name FROM samples WHERE id = sample_id), movement_type, from_location, to_location, reason, created_at
`, tenant.TenantID, input.SampleID, input.MovementType, input.FromLocation, input.ToLocation, input.Reason).Scan(
		&item.ID, &item.TenantID, &item.SampleID, &item.SampleCode, &item.SampleName, &item.MovementType, &item.FromLocation, &item.ToLocation, &item.Reason, &item.CreatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "sample_movement.create", "sample_movement", item.ID, "", item.SampleCode)
	}
	return item, err
}

func (r *Repository) IotDevices(ctx context.Context) ([]IotDevice, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, name, vendor, device_code, COALESCE(instrument_id::text, ''), COALESCE(i.name, ''), online, status, last_seen_at, telemetry::text, notes, created_at, updated_at
FROM iot_devices d
LEFT JOIN instruments i ON i.id = d.instrument_id
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY updated_at DESC, name
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]IotDevice, 0)
	for rows.Next() {
		var item IotDevice
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Name, &item.Vendor, &item.DeviceCode, &item.InstrumentID, &item.InstrumentName, &item.Online, &item.Status, &item.LastSeenAt, &item.Telemetry, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveIotDevice(ctx context.Context, id string, input IotDeviceInput) (IotDevice, error) {
	tenant := TenantFromContext(ctx)
	input.Name = strings.TrimSpace(input.Name)
	input.Vendor = strings.TrimSpace(input.Vendor)
	input.DeviceCode = strings.TrimSpace(input.DeviceCode)
	input.InstrumentID = strings.TrimSpace(input.InstrumentID)
	input.Status = strings.TrimSpace(strings.ToLower(input.Status))
	input.Telemetry = strings.TrimSpace(input.Telemetry)
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Name == "" {
		return IotDevice{}, clientError("device name is required")
	}
	if input.Status == "" {
		if input.Online {
			input.Status = "online"
		} else {
			input.Status = "offline"
		}
	}
	if input.Telemetry == "" {
		input.Telemetry = "{}"
	}
	if !json.Valid([]byte(input.Telemetry)) {
		return IotDevice{}, clientError("telemetry must be valid JSON")
	}
	var item IotDevice
	var err error
	if id == "" {
		err = r.db.QueryRow(ctx, `
INSERT INTO iot_devices (tenant_id, instrument_id, name, vendor, device_code, online, status, last_seen_at, telemetry, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, now(), $8::jsonb, $9)
RETURNING id::text, tenant_id::text, name, vendor, device_code, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), online, status, last_seen_at, telemetry::text, notes, created_at, updated_at
`, tenant.TenantID, optionalID(input.InstrumentID), input.Name, input.Vendor, input.DeviceCode, input.Online, input.Status, input.Telemetry, input.Notes).Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Vendor, &item.DeviceCode, &item.InstrumentID, &item.InstrumentName, &item.Online, &item.Status, &item.LastSeenAt, &item.Telemetry, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
		)
		if err == nil {
			r.audit(ctx, input.Actor, "iot.device.create", "iot_device", item.ID, "", item.Name)
		}
		return item, err
	}
	err = r.db.QueryRow(ctx, `
UPDATE iot_devices
SET instrument_id = $2,
    name = $3,
    vendor = $4,
    device_code = $5,
    online = $6,
    status = $7,
    last_seen_at = now(),
    telemetry = $8::jsonb,
    notes = $9,
    updated_at = now()
WHERE id = $1 AND ($10::boolean OR tenant_id = $11::uuid)
RETURNING id::text, tenant_id::text, name, vendor, device_code, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), online, status, last_seen_at, telemetry::text, notes, created_at, updated_at
`, id, optionalID(input.InstrumentID), input.Name, input.Vendor, input.DeviceCode, input.Online, input.Status, input.Telemetry, input.Notes, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Vendor, &item.DeviceCode, &item.InstrumentID, &item.InstrumentName, &item.Online, &item.Status, &item.LastSeenAt, &item.Telemetry, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, input.Actor, "iot.device.update", "iot_device", item.ID, "", item.Name)
	}
	return item, err
}

func (r *Repository) DeleteIotDevice(ctx context.Context, id string, actor string) (IotDevice, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item IotDevice
	err := r.db.QueryRow(ctx, `
DELETE FROM iot_devices
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, tenant_id::text, name, vendor, device_code, COALESCE(instrument_id::text, ''), COALESCE((SELECT name FROM instruments WHERE id = instrument_id), ''), online, status, last_seen_at, telemetry::text, notes, created_at, updated_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Vendor, &item.DeviceCode, &item.InstrumentID, &item.InstrumentName, &item.Online, &item.Status, &item.LastSeenAt, &item.Telemetry, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == nil {
		r.audit(ctx, actor, "iot.device.delete", "iot_device", item.ID, item.Name, "deleted")
	}
	return item, err
}

func (r *Repository) AssistantQueries(ctx context.Context) ([]AssistantQuery, error) {
	tenant := TenantFromContext(ctx)
	rows, err := r.db.Query(ctx, `
SELECT id::text, tenant_id::text, question, answer, context, created_at
FROM assistant_queries
WHERE ($1::boolean OR tenant_id = $2::uuid)
ORDER BY created_at DESC
LIMIT 50
`, tenant.AllTenants, tenant.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AssistantQuery, 0)
	for rows.Next() {
		var item AssistantQuery
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Question, &item.Answer, &item.Context, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AskAssistant(ctx context.Context, input AssistantQueryInput) (AssistantQuery, error) {
	tenant := TenantFromContext(ctx)
	input.Question = strings.TrimSpace(input.Question)
	input.Context = strings.TrimSpace(input.Context)
	input.Requester = strings.TrimSpace(input.Requester)
	input.RequesterID = strings.TrimSpace(input.RequesterID)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}
	if input.Question == "" {
		return AssistantQuery{}, clientError("question is required")
	}

	settings, err := r.aiAssistantSettingsValue(ctx)
	if err != nil {
		return AssistantQuery{}, err
	}
	if !settings.Enabled {
		return AssistantQuery{}, clientError("AI 助手未启用，请先在后台配置模型 API")
	}
	if settings.APIKey == "" || settings.Model == "" || settings.BaseURL == "" {
		return AssistantQuery{}, clientError("AI 助手模型 API 配置不完整")
	}
	systemContext, err := r.assistantSystemContext(ctx, input.Context)
	if err != nil {
		return AssistantQuery{}, err
	}
	answer, err := r.callAssistantModel(ctx, settings, input.Question, systemContext)
	if err != nil {
		return AssistantQuery{}, WrapClientError("AI 助手调用失败", err)
	}

	var item AssistantQuery
	err = r.db.QueryRow(ctx, `
INSERT INTO assistant_queries (tenant_id, requester_id, requester, question, answer, context)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id::text, tenant_id::text, question, answer, context, created_at
`, tenant.TenantID, optionalID(input.RequesterID), firstNonEmpty(input.Requester, input.Actor), input.Question, answer, input.Context).Scan(
		&item.ID, &item.TenantID, &item.Question, &item.Answer, &item.Context, &item.CreatedAt,
	)
	return item, err
}

func (r *Repository) assistantSystemContext(ctx context.Context, extraContext string) (string, error) {
	tenant := TenantFromContext(ctx)
	var dashboard Dashboard
	if d, err := r.Dashboard(ctx); err == nil {
		dashboard = d
	}
	countQuery := func(sql string, args ...any) int {
		var value int
		if err := r.db.QueryRow(ctx, sql, args...).Scan(&value); err != nil {
			return 0
		}
		return value
	}
	lines := []string{
		"当前机构：" + firstNonEmpty(tenant.TenantName, tenant.TenantID),
		fmt.Sprintf("仪器：活跃 %d 台，今日预约 %d 项，待审批 %d 项，使用中 %d 项，已完成 %d 项。", dashboard.ActiveInstruments, dashboard.TodayReservations, dashboard.PendingApprovals, dashboard.InUseReservations, dashboard.CompletedReservations),
		fmt.Sprintf("资源：低库存 %d 项，待处理申领 %d 条，待处理申购 %d 条。",
			countQuery(`SELECT count(*) FROM materials WHERE ($1::boolean OR tenant_id = $2::uuid) AND status <> 'disabled' AND stock <= warning_line`, tenant.AllTenants, tenant.TenantID),
			countQuery(`SELECT count(*) FROM material_requests WHERE ($1::boolean OR tenant_id = $2::uuid) AND status = 'pending'`, tenant.AllTenants, tenant.TenantID),
			countQuery(`SELECT count(*) FROM material_purchases WHERE ($1::boolean OR tenant_id = $2::uuid) AND status IN ('registered', 'returned')`, tenant.AllTenants, tenant.TenantID),
		),
		fmt.Sprintf("培训：课程 %d 门，授权记录 %d 条。", countQuery(`SELECT count(*) FROM training_courses WHERE ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID), countQuery(`SELECT count(*) FROM training_authorizations WHERE ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID)),
		fmt.Sprintf("空间、样本和物联网：空间 %d 个，样本 %d 条，物联网设备 %d 台。", countQuery(`SELECT count(*) FROM spaces WHERE ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID), countQuery(`SELECT count(*) FROM samples WHERE ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID), countQuery(`SELECT count(*) FROM iot_devices WHERE ($1::boolean OR tenant_id = $2::uuid)`, tenant.AllTenants, tenant.TenantID)),
	}
	extraContext = strings.TrimSpace(extraContext)
	if extraContext != "" {
		lines = append(lines, "用户补充背景："+extraContext)
	}
	return strings.Join(lines, "\n"), nil
}

func (r *Repository) callAssistantModel(ctx context.Context, settings aiAssistantSettingsValue, question string, systemContext string) (string, error) {
	settings = normalizeAIAssistantSettingsValue(settings)
	endpoint, err := assistantChatCompletionsEndpoint(settings.BaseURL)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"model":       settings.Model,
		"temperature": settings.Temperature,
		"max_tokens":  settings.MaxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": settings.SystemPrompt},
			{"role": "system", "content": "系统当前可见数据：\n" + systemContext},
			{"role": "user", "content": question},
		},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(mustJSONBytes(payload)))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+settings.APIKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := r.httpClient().Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", clientErrorf("模型接口返回失败：%s", assistantHTTPErrorMessage(response.StatusCode, raw))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	if decoded.Error.Message != "" {
		return "", clientError(strings.TrimSpace(decoded.Error.Message))
	}
	if len(decoded.Choices) == 0 {
		return "", clientError("模型接口未返回回答")
	}
	answer := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if answer == "" {
		return "", clientError("模型接口返回空回答")
	}
	return answer, nil
}

func assistantChatCompletionsEndpoint(baseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", clientError("AI 助手 API 地址不能为空")
	}
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return "", clientError("AI 助手 API 地址无效")
	}
	if strings.HasSuffix(parsed.Path, "/chat/completions") {
		return baseURL, nil
	}
	return baseURL + "/chat/completions", nil
}

func assistantHTTPErrorMessage(status int, raw []byte) string {
	var decoded struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err == nil && strings.TrimSpace(decoded.Error.Message) != "" {
		if decoded.Error.Code != "" {
			return fmt.Sprintf("status=%d code=%s message=%s", status, decoded.Error.Code, decoded.Error.Message)
		}
		return fmt.Sprintf("status=%d message=%s", status, decoded.Error.Message)
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return fmt.Sprintf("status=%d", status)
	}
	if len(text) > 500 {
		text = text[:500]
	}
	return fmt.Sprintf("status=%d body=%s", status, text)
}

func (r *Repository) DeleteAssistantQuery(ctx context.Context, id string, actor string) (AssistantQuery, error) {
	tenant := TenantFromContext(ctx)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	var item AssistantQuery
	err := r.db.QueryRow(ctx, `
DELETE FROM assistant_queries
WHERE id = $1 AND ($2::boolean OR tenant_id = $3::uuid)
RETURNING id::text, tenant_id::text, question, answer, context, created_at
`, id, tenant.AllTenants, tenant.TenantID).Scan(&item.ID, &item.TenantID, &item.Question, &item.Answer, &item.Context, &item.CreatedAt)
	if err == nil {
		r.audit(ctx, actor, "assistant.query.delete", "assistant_query", item.ID, item.Question, "deleted")
	}
	return item, err
}
