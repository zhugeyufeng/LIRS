package store

import "time"

type TrainingCourse struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenantId,omitempty"`
	Title              string    `json:"title"`
	Category           string    `json:"category"`
	InstrumentID       string    `json:"instrumentId,omitempty"`
	InstrumentName     string    `json:"instrumentName,omitempty"`
	Instructor         string    `json:"instructor"`
	DeliveryMode       string    `json:"deliveryMode"`
	DurationHours      float64   `json:"durationHours"`
	RequiredForBooking bool      `json:"requiredForBooking"`
	Status             string    `json:"status"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type TrainingCourseInput struct {
	Title              string  `json:"title"`
	Category           string  `json:"category"`
	InstrumentID       string  `json:"instrumentId"`
	Instructor         string  `json:"instructor"`
	DeliveryMode       string  `json:"deliveryMode"`
	DurationHours      float64 `json:"durationHours"`
	RequiredForBooking bool    `json:"requiredForBooking"`
	Status             string  `json:"status"`
	Description        string  `json:"description"`
	Actor              string  `json:"actor"`
}

type TrainingAuthorization struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	UserID         string    `json:"userId,omitempty"`
	UserName       string    `json:"userName"`
	CourseID       string    `json:"courseId,omitempty"`
	CourseTitle    string    `json:"courseTitle"`
	InstrumentID   string    `json:"instrumentId,omitempty"`
	InstrumentName string    `json:"instrumentName,omitempty"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expiresAt"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TrainingAuthorizationInput struct {
	UserID       string    `json:"userId"`
	UserName     string    `json:"userName"`
	CourseID     string    `json:"courseId"`
	InstrumentID string    `json:"instrumentId"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Notes        string    `json:"notes"`
	Actor        string    `json:"actor"`
}

type TrainingQuestion struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenantId,omitempty"`
	Title         string    `json:"title"`
	QuestionType  string    `json:"questionType"`
	Options       string    `json:"options"`
	CorrectAnswer string    `json:"correctAnswer"`
	Explanation   string    `json:"explanation"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type TrainingQuestionInput struct {
	Title         string `json:"title"`
	QuestionType  string `json:"questionType"`
	Options       string `json:"options"`
	CorrectAnswer string `json:"correctAnswer"`
	Explanation   string `json:"explanation"`
	Status        string `json:"status"`
	Actor         string `json:"actor"`
}

type TrainingExam struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	UserID      string    `json:"userId,omitempty"`
	UserName    string    `json:"userName"`
	CourseID    string    `json:"courseId,omitempty"`
	CourseTitle string    `json:"courseTitle"`
	Score       float64   `json:"score"`
	Passed      bool      `json:"passed"`
	Answers     string    `json:"answers"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
	ExamAt      time.Time `json:"examAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TrainingExamInput struct {
	UserID   string    `json:"userId"`
	UserName string    `json:"userName"`
	CourseID string    `json:"courseId"`
	Score    float64   `json:"score"`
	Passed   bool      `json:"passed"`
	Answers  string    `json:"answers"`
	Status   string    `json:"status"`
	Notes    string    `json:"notes"`
	ExamAt   time.Time `json:"examAt"`
	Actor    string    `json:"actor"`
}

type TrainingPractical struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	UserID         string    `json:"userId,omitempty"`
	UserName       string    `json:"userName"`
	InstrumentID   string    `json:"instrumentId,omitempty"`
	InstrumentName string    `json:"instrumentName,omitempty"`
	Assessor       string    `json:"assessor"`
	Score          float64   `json:"score"`
	Result         string    `json:"result"`
	Notes          string    `json:"notes"`
	AssessmentAt   time.Time `json:"assessmentAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TrainingPracticalInput struct {
	UserID       string    `json:"userId"`
	UserName     string    `json:"userName"`
	InstrumentID string    `json:"instrumentId"`
	Assessor     string    `json:"assessor"`
	Score        float64   `json:"score"`
	Result       string    `json:"result"`
	Notes        string    `json:"notes"`
	AssessmentAt time.Time `json:"assessmentAt"`
	Actor        string    `json:"actor"`
}

type TrainingRule struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenantId,omitempty"`
	InstrumentID    string    `json:"instrumentId,omitempty"`
	InstrumentName  string    `json:"instrumentName,omitempty"`
	RequireTraining bool      `json:"requireTraining"`
	RequireExam     bool      `json:"requireExam"`
	RequireApproval bool      `json:"requireApproval"`
	MinScore        float64   `json:"minScore"`
	Status          string    `json:"status"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type TrainingRuleInput struct {
	InstrumentID    string  `json:"instrumentId"`
	RequireTraining bool    `json:"requireTraining"`
	RequireExam     bool    `json:"requireExam"`
	RequireApproval bool    `json:"requireApproval"`
	MinScore        float64 `json:"minScore"`
	Status          string  `json:"status"`
	Notes           string  `json:"notes"`
	Actor           string  `json:"actor"`
}

type BusinessConfig struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	Domain      string    `json:"domain"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Scope       string    `json:"scope"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	ConfigJSON  string    `json:"configJson"`
	UpdatedBy   string    `json:"updatedBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type BusinessConfigInput struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
	Description string `json:"description"`
	ConfigJSON  string `json:"configJson"`
	Actor       string `json:"actor"`
}

type Space struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenantId,omitempty"`
	Name               string    `json:"name"`
	Kind               string    `json:"kind"`
	Department         string    `json:"department"`
	Location           string    `json:"location"`
	Capacity           int       `json:"capacity"`
	Status             string    `json:"status"`
	AccessControlPoint string    `json:"accessControlPoint"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type SpaceInput struct {
	Name               string `json:"name"`
	Kind               string `json:"kind"`
	Department         string `json:"department"`
	Location           string `json:"location"`
	Capacity           int    `json:"capacity"`
	Status             string `json:"status"`
	AccessControlPoint string `json:"accessControlPoint"`
	Description        string `json:"description"`
	Actor              string `json:"actor"`
}

type SpaceReservation struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	SpaceID     string    `json:"spaceId"`
	SpaceName   string    `json:"spaceName"`
	RequesterID string    `json:"requesterId,omitempty"`
	Requester   string    `json:"requester"`
	Purpose     string    `json:"purpose"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SpaceReservationInput struct {
	SpaceID     string    `json:"spaceId"`
	RequesterID string    `json:"requesterId"`
	Requester   string    `json:"requester"`
	Purpose     string    `json:"purpose"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Actor       string    `json:"actor"`
}

type Sample struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenantId,omitempty"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	OwnerID          string    `json:"ownerId,omitempty"`
	OwnerName        string    `json:"ownerName"`
	Department       string    `json:"department"`
	GroupName        string    `json:"groupName"`
	Location         string    `json:"location"`
	Status           string    `json:"status"`
	HazardLevel      string    `json:"hazardLevel"`
	StorageCondition string    `json:"storageCondition"`
	Description      string    `json:"description"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SampleInput struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	OwnerID          string `json:"ownerId"`
	OwnerName        string `json:"ownerName"`
	Department       string `json:"department"`
	GroupName        string `json:"groupName"`
	Location         string `json:"location"`
	Status           string `json:"status"`
	HazardLevel      string `json:"hazardLevel"`
	StorageCondition string `json:"storageCondition"`
	Description      string `json:"description"`
	Actor            string `json:"actor"`
}

type SampleMovement struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId,omitempty"`
	SampleID     string    `json:"sampleId"`
	SampleCode   string    `json:"sampleCode"`
	SampleName   string    `json:"sampleName"`
	MovementType string    `json:"movementType"`
	FromLocation string    `json:"fromLocation"`
	ToLocation   string    `json:"toLocation"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SampleMovementInput struct {
	SampleID     string `json:"sampleId"`
	MovementType string `json:"movementType"`
	FromLocation string `json:"fromLocation"`
	ToLocation   string `json:"toLocation"`
	Reason       string `json:"reason"`
	Actor        string `json:"actor"`
}

type LimsTask struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	SampleID       string    `json:"sampleId,omitempty"`
	SampleCode     string    `json:"sampleCode,omitempty"`
	InstrumentID   string    `json:"instrumentId,omitempty"`
	InstrumentName string    `json:"instrumentName,omitempty"`
	Title          string    `json:"title"`
	AssayType      string    `json:"assayType"`
	Priority       string    `json:"priority"`
	Status         string    `json:"status"`
	RequesterID    string    `json:"requesterId,omitempty"`
	RequesterName  string    `json:"requesterName"`
	DueAt          time.Time `json:"dueAt"`
	ResultSummary  string    `json:"resultSummary"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type LimsTaskInput struct {
	RequesterID   string    `json:"requesterId"`
	SampleID      string    `json:"sampleId"`
	InstrumentID  string    `json:"instrumentId"`
	Title         string    `json:"title"`
	AssayType     string    `json:"assayType"`
	Priority      string    `json:"priority"`
	Status        string    `json:"status"`
	RequesterName string    `json:"requesterName"`
	DueAt         time.Time `json:"dueAt"`
	ResultSummary string    `json:"resultSummary"`
	Actor         string    `json:"actor"`
}

type ElnRecord struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenantId,omitempty"`
	Title           string    `json:"title"`
	AuthorID        string    `json:"authorId,omitempty"`
	AuthorName      string    `json:"authorName"`
	Project         string    `json:"project"`
	LinkedTaskID    string    `json:"linkedTaskId,omitempty"`
	LinkedTaskTitle string    `json:"linkedTaskTitle,omitempty"`
	Content         string    `json:"content"`
	Status          string    `json:"status"`
	SignedAt        time.Time `json:"signedAt"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ElnRecordInput struct {
	AuthorID     string `json:"authorId"`
	Title        string `json:"title"`
	Project      string `json:"project"`
	LinkedTaskID string `json:"linkedTaskId"`
	Content      string `json:"content"`
	Status       string `json:"status"`
	Actor        string `json:"actor"`
	AuthorName   string `json:"authorName"`
}

type IotDevice struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	Name           string    `json:"name"`
	Vendor         string    `json:"vendor"`
	DeviceCode     string    `json:"deviceCode"`
	InstrumentID   string    `json:"instrumentId,omitempty"`
	InstrumentName string    `json:"instrumentName,omitempty"`
	Online         bool      `json:"online"`
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
	Telemetry      string    `json:"telemetry"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type IotDeviceInput struct {
	Name         string `json:"name"`
	Vendor       string `json:"vendor"`
	DeviceCode   string `json:"deviceCode"`
	InstrumentID string `json:"instrumentId"`
	Online       bool   `json:"online"`
	Status       string `json:"status"`
	Telemetry    string `json:"telemetry"`
	Notes        string `json:"notes"`
	Actor        string `json:"actor"`
}

type AssistantQuery struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId,omitempty"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Context   string    `json:"context"`
	CreatedAt time.Time `json:"createdAt"`
}

type AssistantQueryInput struct {
	RequesterID string `json:"requesterId"`
	Requester   string `json:"requester"`
	Question    string `json:"question"`
	Context     string `json:"context"`
	Actor       string `json:"actor"`
}
