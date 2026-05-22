package httpapi

import (
	"context"
	"time"

	"lirs/apps/server/internal/store"
)

type repository interface {
	systemRepository
	notificationSettingsRepository
	instrumentRepository
	extensionRepository
	reservationRepository
	userRepository
	notificationRepository
	financeRepository
	authFlowRepository
	materialRepository
	procurementRepository
}

type authRepository interface {
	CurrentUser(ctx context.Context, token string) (store.User, error)
}

type tenantReaderRepository interface {
	authRepository
	Tenants(ctx context.Context) ([]store.Tenant, error)
}

type systemRepository interface {
	Health(ctx context.Context) error
	Dashboard(ctx context.Context) (store.Dashboard, error)
	Tenants(ctx context.Context) ([]store.Tenant, error)
	SaveTenant(ctx context.Context, id string, input store.TenantInput) (store.Tenant, error)
	FooterSettings(ctx context.Context) (store.FooterSettings, error)
	SaveFooterSettings(ctx context.Context, input store.FooterSettingsInput) (store.FooterSettings, error)
	CopySettings(ctx context.Context) (store.CopySettings, error)
	SaveCopySettings(ctx context.Context, input store.CopySettingsInput) (store.CopySettings, error)
	AuditEvents(ctx context.Context) ([]store.AuditEvent, error)
	Operations(ctx context.Context) (store.Operations, error)
}

type notificationSettingsRepository interface {
	NotificationChannelSettings(ctx context.Context) (store.NotificationChannelSettings, error)
	SaveGraphMailSettings(ctx context.Context, input store.GraphMailSettingsInput) (store.GraphMailSettings, error)
	TestGraphMailSettings(ctx context.Context, input store.GraphMailTestInput) (store.GraphMailTestResult, error)
	SaveWeChatSettings(ctx context.Context, input store.WeChatSettingsInput) (store.WeChatSettings, error)
	DingTalkSettings(ctx context.Context) (store.DingTalkSettings, error)
	SaveDingTalkSettings(ctx context.Context, input store.DingTalkSettingsInput) (store.DingTalkSettings, error)
	TestDingTalkSettings(ctx context.Context, input store.DingTalkTestInput) (store.DingTalkTestResult, error)
	HandleDingTalkEventCallback(ctx context.Context, input store.DingTalkEventCallbackInput) (store.DingTalkEventCallbackResponse, error)
	AccessControlSettings(ctx context.Context) (store.AccessControlSettings, error)
	SaveAccessControlSettings(ctx context.Context, input store.AccessControlSettingsInput) (store.AccessControlSettings, error)
	AIAssistantSettings(ctx context.Context) (store.AIAssistantSettings, error)
	SaveAIAssistantSettings(ctx context.Context, input store.AIAssistantSettingsInput) (store.AIAssistantSettings, error)
}

type instrumentRepository interface {
	Instruments(ctx context.Context, filter store.InstrumentFilter) ([]store.Instrument, error)
	Instrument(ctx context.Context, id string) (store.Instrument, error)
	InstrumentSlots(ctx context.Context, id string, start time.Time, days int) ([]store.Slot, error)
	SaveInstrument(ctx context.Context, id string, input store.InstrumentInput) (store.Instrument, error)
	DeleteInstrument(ctx context.Context, id string, actor string) (store.Instrument, error)
}

type extensionRepository interface {
	TrainingCourses(ctx context.Context) ([]store.TrainingCourse, error)
	SaveTrainingCourse(ctx context.Context, id string, input store.TrainingCourseInput) (store.TrainingCourse, error)
	TrainingAuthorizations(ctx context.Context) ([]store.TrainingAuthorization, error)
	SaveTrainingAuthorization(ctx context.Context, id string, input store.TrainingAuthorizationInput) (store.TrainingAuthorization, error)
	TrainingQuestions(ctx context.Context) ([]store.TrainingQuestion, error)
	SaveTrainingQuestion(ctx context.Context, id string, input store.TrainingQuestionInput) (store.TrainingQuestion, error)
	TrainingExams(ctx context.Context) ([]store.TrainingExam, error)
	SaveTrainingExam(ctx context.Context, id string, input store.TrainingExamInput) (store.TrainingExam, error)
	TrainingPracticals(ctx context.Context) ([]store.TrainingPractical, error)
	SaveTrainingPractical(ctx context.Context, id string, input store.TrainingPracticalInput) (store.TrainingPractical, error)
	TrainingRules(ctx context.Context) ([]store.TrainingRule, error)
	SaveTrainingRule(ctx context.Context, id string, input store.TrainingRuleInput) (store.TrainingRule, error)
	BusinessConfigs(ctx context.Context, domain string, kind string) ([]store.BusinessConfig, error)
	SaveBusinessConfig(ctx context.Context, domain string, kind string, id string, input store.BusinessConfigInput) (store.BusinessConfig, error)
	Spaces(ctx context.Context) ([]store.Space, error)
	SaveSpace(ctx context.Context, id string, input store.SpaceInput) (store.Space, error)
	SpaceReservations(ctx context.Context) ([]store.SpaceReservation, error)
	CreateSpaceReservation(ctx context.Context, input store.SpaceReservationInput) (store.SpaceReservation, error)
	Samples(ctx context.Context) ([]store.Sample, error)
	SaveSample(ctx context.Context, id string, input store.SampleInput) (store.Sample, error)
	SampleMovements(ctx context.Context) ([]store.SampleMovement, error)
	CreateSampleMovement(ctx context.Context, input store.SampleMovementInput) (store.SampleMovement, error)
	IotDevices(ctx context.Context) ([]store.IotDevice, error)
	SaveIotDevice(ctx context.Context, id string, input store.IotDeviceInput) (store.IotDevice, error)
	DeleteIotDevice(ctx context.Context, id string, actor string) (store.IotDevice, error)
	AssistantQueries(ctx context.Context) ([]store.AssistantQuery, error)
	AskAssistant(ctx context.Context, input store.AssistantQueryInput) (store.AssistantQuery, error)
	DeleteAssistantQuery(ctx context.Context, id string, actor string) (store.AssistantQuery, error)
}

type reservationRepository interface {
	Reservation(ctx context.Context, id string) (store.Reservation, error)
	Reservations(ctx context.Context) ([]store.Reservation, error)
	CreateReservation(ctx context.Context, input store.ReservationInput) (store.Reservation, error)
	CreateReservationBatch(ctx context.Context, input store.ReservationBatchInput) ([]store.Reservation, error)
	ApproveReservation(ctx context.Context, id string, approved bool, actor string, comment string) (store.Reservation, error)
	CheckInReservation(ctx context.Context, id string) (store.Reservation, error)
	CompleteReservation(ctx context.Context, id string) (store.Reservation, error)
	CancelReservation(ctx context.Context, id string, reason string, bypassCutoff bool) (store.Reservation, error)
	MaintenanceOrders(ctx context.Context) ([]store.MaintenanceOrder, error)
	CreateMaintenanceOrder(ctx context.Context, input store.MaintenanceInput) (store.MaintenanceOrder, error)
	StartMaintenanceOrder(ctx context.Context, id string, actor string) (store.MaintenanceOrder, error)
	CancelMaintenanceOrder(ctx context.Context, id string, reason string, actor string) (store.MaintenanceOrder, error)
	CompleteMaintenanceOrder(ctx context.Context, id string, result string, actor string) (store.MaintenanceOrder, error)
}

type userRepository interface {
	Users(ctx context.Context) ([]store.User, error)
	CreateUser(ctx context.Context, input store.UserCreateInput) (store.User, error)
	ReviewUser(ctx context.Context, id string, input store.UserReviewInput) (store.User, error)
	SaveUserMembership(ctx context.Context, id string, input store.UserMembershipInput) (store.User, error)
	DeleteUser(ctx context.Context, id string, actor string) (store.User, error)
	OrganizationUnits(ctx context.Context, kind string) ([]store.OrganizationUnit, error)
	SaveOrganizationUnit(ctx context.Context, id string, input store.OrganizationUnitInput) (store.OrganizationUnit, error)
	DeleteOrganizationUnit(ctx context.Context, id string, actor string) (store.OrganizationUnit, error)
}

type notificationRepository interface {
	Notifications(ctx context.Context, actor store.Actor) ([]store.Notification, error)
	MarkNotificationRead(ctx context.Context, id string, actor store.Actor) (store.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, actor store.Actor) (int, error)
	DeleteNotification(ctx context.Context, id string, actor string) (store.Notification, error)
	Announce(ctx context.Context, input store.AnnouncementInput) (store.Notification, error)
	UpdateNotification(ctx context.Context, id string, input store.AnnouncementInput) (store.Notification, error)
}

type financeRepository interface {
	Ledger(ctx context.Context, actor store.Actor) ([]store.LedgerEntry, error)
	AdjustLedger(ctx context.Context, input store.LedgerAdjustmentInput) (store.LedgerEntry, error)
	FinancialAccounts(ctx context.Context, actor store.Actor) ([]store.FinancialAccount, error)
	SaveFinancialAccount(ctx context.Context, id string, input store.FinancialAccountInput) (store.FinancialAccount, error)
}

type authFlowRepository interface {
	Register(ctx context.Context, input store.RegisterInput) (store.User, error)
	RequestEmailVerificationCode(ctx context.Context, input store.EmailVerificationCodeInput) (store.EmailVerificationCodeResponse, error)
	Login(ctx context.Context, input store.LoginInput) (store.AuthResponse, error)
	DingTalkQuickLogin(ctx context.Context, input store.DingTalkQuickLoginInput) (store.AuthResponse, error)
	DingTalkWebLoginIntent(ctx context.Context, input store.DingTalkWebLoginIntentInput) (store.DingTalkWebLoginIntent, error)
	DingTalkWebLogin(ctx context.Context, input store.DingTalkWebLoginInput) (store.DingTalkWebLoginResult, error)
	BindDingTalkLoginToExistingUser(ctx context.Context, input store.DingTalkLoginBindExistingInput) (store.AuthResponse, error)
	CurrentUser(ctx context.Context, token string) (store.User, error)
	Logout(ctx context.Context, token string) error
	LogoutAll(ctx context.Context, userID string) error
	VerifyEmail(ctx context.Context, token string) (store.User, error)
	UpdateCurrentUserProfile(ctx context.Context, id string, input store.UserProfileInput) (store.User, error)
	CurrentUserDingTalkBinding(ctx context.Context, id string) (store.DingTalkBinding, error)
	BindCurrentUserDingTalk(ctx context.Context, id string, input store.DingTalkBindingInput) (store.DingTalkBinding, error)
	UnbindCurrentUserDingTalk(ctx context.Context, id string, actor string) (store.DingTalkBinding, error)
	ChangePassword(ctx context.Context, id string, input store.PasswordChangeInput) error
}

type materialRepository interface {
	Materials(ctx context.Context) ([]store.Material, error)
	Material(ctx context.Context, id string) (store.Material, error)
	MaterialByQRCode(ctx context.Context, code string) (store.Material, error)
	DeleteMaterial(ctx context.Context, id string, actor string) (store.Material, error)
	MaterialCategories(ctx context.Context) ([]store.MaterialCategory, error)
	SaveMaterialCategory(ctx context.Context, id string, input store.MaterialCategoryInput) (store.MaterialCategory, error)
	DeleteMaterialCategory(ctx context.Context, id string, actor string) (store.MaterialCategory, error)
	MaterialRequest(ctx context.Context, id string) (store.MaterialRequest, error)
	InventoryLedger(ctx context.Context) ([]store.InventoryLedgerEntry, error)
	SaveMaterial(ctx context.Context, id string, input store.MaterialInput) (store.Material, error)
	ImportMaterials(ctx context.Context, input store.MaterialImportInput) (store.MaterialImportResult, error)
	ImportMaterialsCSV(ctx context.Context, content string, actor string) (store.MaterialImportResult, error)
	MaterialAnalytics(ctx context.Context) (store.MaterialAnalytics, error)
	MaterialAlertActions(ctx context.Context) ([]store.MaterialAlertAction, error)
	CreateMaterialAlertAction(ctx context.Context, materialID string, input store.MaterialAlertActionInput) (store.MaterialAlertAction, error)
	AdjustMaterialStock(ctx context.Context, id string, input store.StockAdjustmentInput) (store.Material, error)
	MaterialRequests(ctx context.Context) ([]store.MaterialRequest, error)
	MaterialRequestsForMonth(ctx context.Context, month string) ([]store.MaterialRequestExportRow, error)
	CreateMaterialRequest(ctx context.Context, input store.MaterialRequestInput) (store.MaterialRequest, error)
	ApproveMaterialRequest(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialRequest, error)
	OutboundMaterialRequest(ctx context.Context, id string, actor string) (store.MaterialRequest, error)
	CancelMaterialRequest(ctx context.Context, id string, actor string) (store.MaterialRequest, error)
	MaterialDamage(ctx context.Context, id string) (store.MaterialDamage, error)
	MaterialDamages(ctx context.Context) ([]store.MaterialDamage, error)
	CreateMaterialDamage(ctx context.Context, input store.MaterialDamageInput) (store.MaterialDamage, error)
	ApproveMaterialDamage(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialDamage, error)
	ProcessMaterialDamage(ctx context.Context, id string, actor string) (store.MaterialDamage, error)
	CancelMaterialDamage(ctx context.Context, id string, actor string) (store.MaterialDamage, error)
}

type procurementRepository interface {
	ProcurementProjects(ctx context.Context) ([]store.ProcurementProject, error)
	SaveProcurementProject(ctx context.Context, id string, input store.ProcurementProjectInput) (store.ProcurementProject, error)
	DeleteProcurementProject(ctx context.Context, id string, actor string) (store.ProcurementProject, error)
	PurchasableMaterials(ctx context.Context) ([]store.PurchasableMaterial, error)
	SavePurchasableMaterial(ctx context.Context, id string, input store.PurchasableMaterialInput) (store.PurchasableMaterial, error)
	DeletePurchasableMaterial(ctx context.Context, id string, actor string) (store.PurchasableMaterial, error)
	ImportPurchasableMaterials(ctx context.Context, input store.PurchasableMaterialImportInput) (store.MaterialImportResult, error)
	MaterialPurchase(ctx context.Context, id string) (store.MaterialPurchase, error)
	MaterialPurchases(ctx context.Context) ([]store.MaterialPurchase, error)
	MaterialPurchaseMonthlyConfirmations(ctx context.Context) ([]store.MaterialPurchaseMonthlyConfirmation, error)
	ConfirmMaterialPurchaseMonth(ctx context.Context, month string, actor string) (store.MaterialPurchaseMonthlyConfirmation, error)
	CreateMaterialPurchase(ctx context.Context, input store.MaterialPurchaseInput) (store.MaterialPurchase, error)
	UpdateMaterialPurchase(ctx context.Context, id string, input store.MaterialPurchaseUpdateInput) (store.MaterialPurchase, error)
	ApproveMaterialPurchase(ctx context.Context, id string, approved bool, actor string, comment string) (store.MaterialPurchase, error)
	ReturnMaterialPurchase(ctx context.Context, id string, actor string, comment string) (store.MaterialPurchase, error)
	MarkMaterialPurchaseOrdered(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
	ReceiveMaterialPurchase(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
	CancelMaterialPurchase(ctx context.Context, id string, actor string) (store.MaterialPurchase, error)
}
