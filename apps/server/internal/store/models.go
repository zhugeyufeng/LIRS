package store

import "time"

type Instrument struct {
	ID                   string  `json:"id"`
	TenantID             string  `json:"tenantId,omitempty"`
	Name                 string  `json:"name"`
	Category             string  `json:"category"`
	Department           string  `json:"department"`
	GroupName            string  `json:"groupName"`
	Status               string  `json:"status"`
	Location             string  `json:"location"`
	HourlyRate           float64 `json:"hourlyRate"`
	Brand                string  `json:"brand"`
	Model                string  `json:"model"`
	AssetCode            string  `json:"assetCode"`
	AccessControlEnabled bool    `json:"accessControlEnabled"`
	AccessControlGroup   string  `json:"accessControlGroup"`
	AccessControlPoint   string  `json:"accessControlPoint"`
	Description          string  `json:"description"`
	TechnicalSpecs       string  `json:"technicalSpecs"`
	BookingRule          string  `json:"bookingRule"`
	MaintenanceSummary   string  `json:"maintenanceSummary"`
	MaxBookingHours      int     `json:"maxBookingHours"`
	MinAdvanceHours      int     `json:"minAdvanceHours"`
	CancelCutoffHours    int     `json:"cancelCutoffHours"`
	CheckinWindowMins    int     `json:"checkinWindowMinutes"`
	BookingWindowDays    int     `json:"bookingWindowDays"`
	BookingIntervalHours int     `json:"bookingIntervalHours"`
	ServiceStartHour     int     `json:"serviceStartHour"`
	ServiceEndHour       int     `json:"serviceEndHour"`
	UsageCount           int     `json:"usageCount"`
}

type Reservation struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	UserID         string    `json:"userId,omitempty"`
	InstrumentID   string    `json:"instrumentId"`
	InstrumentName string    `json:"instrumentName"`
	UserName       string    `json:"userName"`
	GroupName      string    `json:"groupName"`
	Purpose        string    `json:"purpose"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	Status         string    `json:"status"`
	Fee            float64   `json:"fee"`
}

type User struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenantId"`
	TenantName      string `json:"tenantName"`
	TenantCode      string `json:"tenantCode"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Department      string `json:"department"`
	GroupName       string `json:"groupName"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	EmailVerified   bool   `json:"emailVerified"`
	DingTalkUserID  string `json:"dingTalkUserId"`
	DingTalkUnionID string `json:"dingTalkUnionId"`
	DingTalkName    string `json:"dingTalkName"`
	DingTalkBound   bool   `json:"dingTalkBound"`
	FinanceEnabled  bool   `json:"financeEnabled"`
	AuthEpoch       int    `json:"authEpoch"`
}

type Tenant struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	FinanceEnabled bool      `json:"financeEnabled"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type TenantInput struct {
	Name           string `json:"name"`
	Code           string `json:"code"`
	FinanceEnabled bool   `json:"financeEnabled"`
	Status         string `json:"status"`
	Actor          string `json:"actor"`
}

type Notification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId,omitempty"`
	GroupName   string    `json:"groupName,omitempty"`
	Department  string    `json:"department,omitempty"`
	TargetScope string    `json:"targetScope"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Level       string    `json:"level"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"createdAt"`
}

type LedgerEntry struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId,omitempty"`
	UserName      string    `json:"userName,omitempty"`
	ReservationID string    `json:"reservationId,omitempty"`
	GroupName     string    `json:"groupName"`
	Description   string    `json:"description"`
	Amount        float64   `json:"amount"`
	EntryType     string    `json:"entryType"`
	ReferenceID   string    `json:"referenceId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Dashboard struct {
	TodayReservations     int     `json:"todayReservations"`
	PendingApprovals      int     `json:"pendingApprovals"`
	InUseReservations     int     `json:"inUseReservations"`
	CompletedReservations int     `json:"completedReservations"`
	FulfillmentRate       float64 `json:"fulfillmentRate"`
	ActiveInstruments     int     `json:"activeInstruments"`
	MonthlyRevenue        float64 `json:"monthlyRevenue"`
}

type FooterSection struct {
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

type FooterSettings struct {
	Key          string          `json:"key"`
	BrandName    string          `json:"brandName"`
	BrandTagline string          `json:"brandTagline"`
	Description  string          `json:"description"`
	Sections     []FooterSection `json:"sections"`
	Copyright    string          `json:"copyright"`
	UpdatedBy    string          `json:"updatedBy"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type FooterSettingsInput struct {
	BrandName    string          `json:"brandName"`
	BrandTagline string          `json:"brandTagline"`
	Description  string          `json:"description"`
	Sections     []FooterSection `json:"sections"`
	Copyright    string          `json:"copyright"`
	Actor        string          `json:"actor"`
}

type CopyEntry struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
}

type CopySettings struct {
	Key       string      `json:"key"`
	Entries   []CopyEntry `json:"entries"`
	UpdatedBy string      `json:"updatedBy"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type CopySettingsInput struct {
	Entries []CopyEntry `json:"entries"`
	Actor   string      `json:"actor"`
}

type SMTPSettings struct {
	Enabled            bool      `json:"enabled"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	Username           string    `json:"username"`
	FromEmail          string    `json:"fromEmail"`
	FromName           string    `json:"fromName"`
	PasswordConfigured bool      `json:"passwordConfigured"`
	UpdatedBy          string    `json:"updatedBy"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type SMTPSettingsInput struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"fromEmail"`
	FromName  string `json:"fromName"`
	Actor     string `json:"actor"`
}

type WeChatSettings struct {
	Enabled             bool      `json:"enabled"`
	AccountType         string    `json:"accountType"`
	AppID               string    `json:"appId"`
	ServiceAccountName  string    `json:"serviceAccountName"`
	TemplateID          string    `json:"templateId"`
	Token               string    `json:"token"`
	EncodingAESKey      string    `json:"encodingAesKey"`
	AppSecretConfigured bool      `json:"appSecretConfigured"`
	UpdatedBy           string    `json:"updatedBy"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type WeChatSettingsInput struct {
	Enabled            bool   `json:"enabled"`
	AccountType        string `json:"accountType"`
	AppID              string `json:"appId"`
	AppSecret          string `json:"appSecret"`
	ServiceAccountName string `json:"serviceAccountName"`
	TemplateID         string `json:"templateId"`
	Token              string `json:"token"`
	EncodingAESKey     string `json:"encodingAesKey"`
	Actor              string `json:"actor"`
}

type DingTalkSettings struct {
	SchemaVersion          int       `json:"schemaVersion"`
	Enabled                bool      `json:"enabled"`
	ClientID               string    `json:"clientId"`
	ClientSecret           string    `json:"clientSecret,omitempty"`
	CorpID                 string    `json:"corpId"`
	RobotCode              string    `json:"robotCode"`
	OAuthRedirectURI       string    `json:"oauthRedirectUri"`
	EventCallbackURL       string    `json:"eventCallbackUrl"`
	EventAesKey            string    `json:"eventAesKey,omitempty"`
	EventToken             string    `json:"eventToken,omitempty"`
	ClientSecretConfigured bool      `json:"clientSecretConfigured"`
	EventAesKeyConfigured  bool      `json:"eventAesKeyConfigured"`
	EventTokenConfigured   bool      `json:"eventTokenConfigured"`
	UpdatedBy              string    `json:"updatedBy"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type DingTalkSettingsInput struct {
	SchemaVersion    int    `json:"schemaVersion"`
	Enabled          bool   `json:"enabled"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	CorpID           string `json:"corpId"`
	RobotCode        string `json:"robotCode"`
	OAuthRedirectURI string `json:"oauthRedirectUri"`
	EventCallbackURL string `json:"eventCallbackUrl"`
	EventAesKey      string `json:"eventAesKey"`
	EventToken       string `json:"eventToken"`
	Actor            string `json:"actor"`
}

type DingTalkEventCallbackInput struct {
	TenantID   string `json:"tenantId"`
	TenantCode string `json:"tenantCode"`
	Signature  string `json:"signature"`
	Timestamp  string `json:"timestamp"`
	Nonce      string `json:"nonce"`
	Encrypt    string `json:"encrypt"`
}

type DingTalkEventCallbackResponse struct {
	MsgSignature string `json:"msg_signature"`
	TimeStamp    string `json:"timeStamp"`
	Nonce        string `json:"nonce"`
	Encrypt      string `json:"encrypt"`
}

type NotificationChannelSettings struct {
	SMTP     SMTPSettings     `json:"smtp"`
	WeChat   WeChatSettings   `json:"wechat"`
	DingTalk DingTalkSettings `json:"dingtalk"`
}

type AccessControlSettings struct {
	Enabled                bool      `json:"enabled"`
	Vendor                 string    `json:"vendor"`
	Endpoint               string    `json:"endpoint"`
	ClientID               string    `json:"clientId"`
	AccessGroup            string    `json:"accessGroup"`
	AutoGrantOnApproval    bool      `json:"autoGrantOnApproval"`
	AutoRevokeOnCompletion bool      `json:"autoRevokeOnCompletion"`
	ClientSecretConfigured bool      `json:"clientSecretConfigured"`
	UpdatedBy              string    `json:"updatedBy"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type AccessControlSettingsInput struct {
	Enabled                bool   `json:"enabled"`
	Vendor                 string `json:"vendor"`
	Endpoint               string `json:"endpoint"`
	ClientID               string `json:"clientId"`
	ClientSecret           string `json:"clientSecret"`
	AccessGroup            string `json:"accessGroup"`
	AutoGrantOnApproval    bool   `json:"autoGrantOnApproval"`
	AutoRevokeOnCompletion bool   `json:"autoRevokeOnCompletion"`
	Actor                  string `json:"actor"`
}

type RegisterInput struct {
	TenantID         string `json:"tenantId"`
	TenantCode       string `json:"tenantCode"`
	AccountType      string `json:"accountType"`
	Name             string `json:"name"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Department       string `json:"department"`
	VerificationCode string `json:"verificationCode"`
}

type EmailVerificationCodeInput struct {
	TenantID   string `json:"tenantId"`
	TenantCode string `json:"tenantCode"`
	Email      string `json:"email"`
}

type EmailVerificationCodeResponse struct {
	Sent    bool   `json:"sent"`
	Message string `json:"message"`
}

type LoginInput struct {
	TenantID   string `json:"tenantId"`
	TenantCode string `json:"tenantCode"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Device     string `json:"device"`
}

type DingTalkQuickLoginInput struct {
	TenantID   string `json:"tenantId"`
	TenantCode string `json:"tenantCode"`
	AuthCode   string `json:"authCode"`
	CorpID     string `json:"corpId"`
	Device     string `json:"device"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      User      `json:"user"`
}

type Actor struct {
	UserID         string
	TenantID       string
	TenantName     string
	Name           string
	Email          string
	Department     string
	Role           string
	Status         string
	GroupName      string
	EmailVerified  bool
	FinanceEnabled bool
	AuthEpoch      int
}

type ReservationInput struct {
	InstrumentID   string    `json:"instrumentId"`
	UserID         string    `json:"userId,omitempty"`
	UserName       string    `json:"userName"`
	Purpose        string    `json:"purpose"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	IdempotencyKey string    `json:"idempotencyKey"`
}

type ReservationPeriodInput struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

type ReservationBatchInput struct {
	InstrumentID   string                   `json:"instrumentId"`
	UserID         string                   `json:"userId,omitempty"`
	UserName       string                   `json:"userName"`
	Purpose        string                   `json:"purpose"`
	Periods        []ReservationPeriodInput `json:"periods"`
	IdempotencyKey string                   `json:"idempotencyKey"`
}

type InstrumentInput struct {
	Name                 string  `json:"name"`
	Category             string  `json:"category"`
	Department           string  `json:"department"`
	GroupName            string  `json:"groupName"`
	Status               string  `json:"status"`
	Location             string  `json:"location"`
	HourlyRate           float64 `json:"hourlyRate"`
	Brand                string  `json:"brand"`
	Model                string  `json:"model"`
	AssetCode            string  `json:"assetCode"`
	AccessControlEnabled bool    `json:"accessControlEnabled"`
	AccessControlGroup   string  `json:"accessControlGroup"`
	AccessControlPoint   string  `json:"accessControlPoint"`
	Description          string  `json:"description"`
	TechnicalSpecs       string  `json:"technicalSpecs"`
	BookingRule          string  `json:"bookingRule"`
	MaintenanceSummary   string  `json:"maintenanceSummary"`
	MaxBookingHours      int     `json:"maxBookingHours"`
	MinAdvanceHours      int     `json:"minAdvanceHours"`
	CancelCutoffHours    int     `json:"cancelCutoffHours"`
	CheckinWindowMins    int     `json:"checkinWindowMinutes"`
	BookingWindowDays    int     `json:"bookingWindowDays"`
	BookingIntervalHours int     `json:"bookingIntervalHours"`
	ServiceStartHour     int     `json:"serviceStartHour"`
	ServiceEndHour       int     `json:"serviceEndHour"`
	Actor                string  `json:"actor"`
}

type InstrumentFilter struct {
	Search     string
	Category   string
	Department string
	GroupName  string
	Status     string
	Limit      int
	Offset     int
}

type UserReviewInput struct {
	TenantID   string `json:"tenantId"`
	Role       string `json:"role"`
	GroupName  string `json:"groupName"`
	Department string `json:"department"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Status     string `json:"status"`
	Actor      string `json:"actor"`
	ActorRole  string `json:"-"`
}

type UserMembershipInput struct {
	TenantID   string `json:"tenantId"`
	Role       string `json:"role"`
	GroupName  string `json:"groupName"`
	Department string `json:"department"`
	Status     string `json:"status"`
	Actor      string `json:"actor"`
}

type OrganizationUnit struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	ParentName string    `json:"parentName"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type OrganizationUnitInput struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	ParentName string `json:"parentName"`
	Actor      string `json:"actor"`
}

type UserProfileInput struct {
	Name       *string `json:"name"`
	Phone      *string `json:"phone"`
	Department *string `json:"department"`
	GroupName  *string `json:"groupName"`
	Actor      string  `json:"actor"`
}

type DingTalkBinding struct {
	Bound     bool      `json:"bound"`
	UserID    string    `json:"userId"`
	UnionID   string    `json:"unionId"`
	Name      string    `json:"name"`
	AuthURL   string    `json:"authUrl,omitempty"`
	State     string    `json:"state,omitempty"`
	BoundAt   time.Time `json:"boundAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type DingTalkBindingInput struct {
	AuthCode string `json:"authCode"`
	State    string `json:"state"`
	Actor    string `json:"actor"`
}

type PasswordChangeInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	Actor           string `json:"actor"`
}

type Slot struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

type Material struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	ProductType            string          `json:"productType"`
	Category               string          `json:"category"`
	Subcategory            string          `json:"subcategory"`
	Spec                   string          `json:"spec"`
	Unit                   string          `json:"unit"`
	UnitPrice              float64         `json:"unitPrice"`
	Stock                  int             `json:"stock"`
	WarningLine            int             `json:"warningLine"`
	Supplier               string          `json:"supplier"`
	Manufacturer           string          `json:"manufacturer"`
	BatchNo                string          `json:"batchNo"`
	CatalogNo              string          `json:"catalogNo"`
	CASNo                  string          `json:"casNo"`
	Grade                  string          `json:"grade"`
	Concentration          string          `json:"concentration"`
	ParentMaterialID       string          `json:"parentMaterialId"`
	ParentMaterialName     string          `json:"parentMaterialName"`
	DilutionFactor         string          `json:"dilutionFactor"`
	PreparationMethod      string          `json:"preparationMethod"`
	StorageCondition       string          `json:"storageCondition"`
	StorageRoom            string          `json:"storageRoom"`
	StorageCabinet         string          `json:"storageCabinet"`
	StorageLayer           string          `json:"storageLayer"`
	StorageSlot            string          `json:"storageSlot"`
	TenderContract         string          `json:"tenderContract"`
	ContractNo             string          `json:"contractNo"`
	CertificateURL         string          `json:"certificateUrl"`
	StandardCertificateURL string          `json:"standardCertificateUrl"`
	AttachmentURL          string          `json:"attachmentUrl"`
	QRCode                 string          `json:"qrCode"`
	ExpiresAt              string          `json:"expiresAt"`
	OpenedAt               string          `json:"openedAt"`
	OpenExpireDays         int             `json:"openExpireDays"`
	OpenExpiresAt          string          `json:"openExpiresAt"`
	FreezeThawCount        int             `json:"freezeThawCount"`
	FreezeThawLimit        int             `json:"freezeThawLimit"`
	ApprovalRequired       bool            `json:"approvalRequired"`
	NearExpiryDays         int             `json:"nearExpiryDays"`
	DamagedQuantity        int             `json:"damagedQuantity"`
	Batches                []MaterialBatch `json:"batches"`
	Units                  []MaterialUnit  `json:"units"`
	Status                 string          `json:"status"`
}

type MaterialBatch struct {
	ID         string         `json:"id"`
	MaterialID string         `json:"materialId"`
	BatchNo    string         `json:"batchNo"`
	Quantity   int            `json:"quantity"`
	ExpiresAt  string         `json:"expiresAt"`
	Location   string         `json:"location"`
	Units      []MaterialUnit `json:"units"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type MaterialUnit struct {
	ID         string    `json:"id"`
	MaterialID string    `json:"materialId"`
	BatchID    string    `json:"batchId,omitempty"`
	BatchNo    string    `json:"batchNo,omitempty"`
	UnitCode   string    `json:"unitCode"`
	ExpiresAt  string    `json:"expiresAt"`
	Location   string    `json:"location"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type MaterialInput struct {
	Name                   string  `json:"name"`
	ProductType            string  `json:"productType"`
	Category               string  `json:"category"`
	Subcategory            string  `json:"subcategory"`
	Spec                   string  `json:"spec"`
	Unit                   string  `json:"unit"`
	UnitPrice              float64 `json:"unitPrice"`
	Stock                  int     `json:"stock"`
	WarningLine            int     `json:"warningLine"`
	Supplier               string  `json:"supplier"`
	Manufacturer           string  `json:"manufacturer"`
	BatchNo                string  `json:"batchNo"`
	CatalogNo              string  `json:"catalogNo"`
	CASNo                  string  `json:"casNo"`
	Grade                  string  `json:"grade"`
	Concentration          string  `json:"concentration"`
	ParentMaterialID       string  `json:"parentMaterialId"`
	DilutionFactor         string  `json:"dilutionFactor"`
	PreparationMethod      string  `json:"preparationMethod"`
	StorageCondition       string  `json:"storageCondition"`
	StorageRoom            string  `json:"storageRoom"`
	StorageCabinet         string  `json:"storageCabinet"`
	StorageLayer           string  `json:"storageLayer"`
	StorageSlot            string  `json:"storageSlot"`
	TenderContract         string  `json:"tenderContract"`
	ContractNo             string  `json:"contractNo"`
	CertificateURL         string  `json:"certificateUrl"`
	StandardCertificateURL string  `json:"standardCertificateUrl"`
	AttachmentURL          string  `json:"attachmentUrl"`
	QRCode                 string  `json:"qrCode"`
	ExpiresAt              string  `json:"expiresAt"`
	OpenedAt               string  `json:"openedAt"`
	OpenExpireDays         int     `json:"openExpireDays"`
	FreezeThawCount        int     `json:"freezeThawCount"`
	FreezeThawLimit        int     `json:"freezeThawLimit"`
	ApprovalRequired       bool    `json:"approvalRequired"`
	NearExpiryDays         int     `json:"nearExpiryDays"`
	Status                 string  `json:"status"`
	Actor                  string  `json:"actor"`
}

type StockAdjustmentInput struct {
	ChangeQty int    `json:"changeQty"`
	Reason    string `json:"reason"`
	BatchID   string `json:"batchId"`
	BatchNo   string `json:"batchNo"`
	UnitID    string `json:"unitId"`
	ExpiresAt string `json:"expiresAt"`
	Location  string `json:"location"`
	Actor     string `json:"actor"`
}

type MaterialImportResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

type MaterialCategory struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ParentName   string    `json:"parentName"`
	DisplayOrder int       `json:"displayOrder"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type MaterialCategoryInput struct {
	Name         string `json:"name"`
	ParentName   string `json:"parentName"`
	DisplayOrder int    `json:"displayOrder"`
	Status       string `json:"status"`
	Actor        string `json:"actor"`
}

type MaterialAlertAction struct {
	ID           string    `json:"id"`
	MaterialID   string    `json:"materialId"`
	MaterialName string    `json:"materialName"`
	AlertType    string    `json:"alertType"`
	Action       string    `json:"action"`
	Comment      string    `json:"comment"`
	Actor        string    `json:"actor"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MaterialAlertActionInput struct {
	AlertType string `json:"alertType"`
	Action    string `json:"action"`
	Comment   string `json:"comment"`
	Actor     string `json:"actor"`
}

type MaterialAnalytics struct {
	ProductTotal         int                          `json:"productTotal"`
	StockTotal           int                          `json:"stockTotal"`
	StandardTotal        int                          `json:"standardTotal"`
	TodayUsageTotal      int                          `json:"todayUsageTotal"`
	NearExpiryTotal      int                          `json:"nearExpiryTotal"`
	ExpiredTotal         int                          `json:"expiredTotal"`
	LowStockTotal        int                          `json:"lowStockTotal"`
	DamagedTotal         int                          `json:"damagedTotal"`
	MonthlyConsumption   []MaterialConsumptionPoint   `json:"monthlyConsumption"`
	TopConsumedMaterials []MaterialConsumptionRanking `json:"topConsumedMaterials"`
	DamageByReason       []MaterialDamageReasonStat   `json:"damageByReason"`
	ProductTypeBreakdown []MaterialBreakdown          `json:"productTypeBreakdown"`
	CategoryBreakdown    []MaterialBreakdown          `json:"categoryBreakdown"`
	LatestAlertActions   []MaterialAlertAction        `json:"latestAlertActions"`
}

type MaterialConsumptionPoint struct {
	Month    string `json:"month"`
	Quantity int    `json:"quantity"`
}

type MaterialConsumptionRanking struct {
	MaterialID   string `json:"materialId"`
	MaterialName string `json:"materialName"`
	Quantity     int    `json:"quantity"`
}

type MaterialDamageReasonStat struct {
	Reason   string `json:"reason"`
	Quantity int    `json:"quantity"`
}

type MaterialBreakdown struct {
	Label string `json:"label"`
	Count int    `json:"count"`
	Stock int    `json:"stock"`
}

type InventoryLedgerEntry struct {
	ID           string    `json:"id"`
	MaterialID   string    `json:"materialId"`
	MaterialName string    `json:"materialName"`
	RequestID    string    `json:"requestId,omitempty"`
	PurchaseID   string    `json:"purchaseId,omitempty"`
	DamageID     string    `json:"damageId,omitempty"`
	ChangeQty    int       `json:"changeQty"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MaterialRequest struct {
	ID           string    `json:"id"`
	MaterialID   string    `json:"materialId"`
	MaterialName string    `json:"materialName"`
	RequesterID  string    `json:"requesterId,omitempty"`
	Requester    string    `json:"requester"`
	GroupName    string    `json:"groupName"`
	BatchID      string    `json:"batchId,omitempty"`
	BatchNo      string    `json:"batchNo,omitempty"`
	UnitID       string    `json:"unitId,omitempty"`
	UnitCode     string    `json:"unitCode,omitempty"`
	Quantity     int       `json:"quantity"`
	Purpose      string    `json:"purpose"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MaterialRequestInput struct {
	MaterialID  string `json:"materialId"`
	RequesterID string `json:"requesterId"`
	Requester   string `json:"requester"`
	BatchID     string `json:"batchId"`
	UnitID      string `json:"unitId"`
	Quantity    int    `json:"quantity"`
	Purpose     string `json:"purpose"`
}

type MaterialPurchase struct {
	ID                 string    `json:"id"`
	MaterialID         string    `json:"materialId"`
	MaterialName       string    `json:"materialName"`
	RequesterID        string    `json:"requesterId,omitempty"`
	Requester          string    `json:"requester"`
	GroupName          string    `json:"groupName"`
	Quantity           int       `json:"quantity"`
	EstimatedUnitPrice float64   `json:"estimatedUnitPrice"`
	Supplier           string    `json:"supplier"`
	Reason             string    `json:"reason"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
}

type MaterialPurchaseInput struct {
	MaterialID         string  `json:"materialId"`
	RequesterID        string  `json:"requesterId"`
	Requester          string  `json:"requester"`
	Quantity           int     `json:"quantity"`
	EstimatedUnitPrice float64 `json:"estimatedUnitPrice"`
	Supplier           string  `json:"supplier"`
	Reason             string  `json:"reason"`
}

type MaterialDamage struct {
	ID            string    `json:"id"`
	MaterialID    string    `json:"materialId"`
	MaterialName  string    `json:"materialName"`
	ReporterID    string    `json:"reporterId,omitempty"`
	Reporter      string    `json:"reporter"`
	GroupName     string    `json:"groupName"`
	BatchID       string    `json:"batchId,omitempty"`
	BatchNo       string    `json:"batchNo,omitempty"`
	UnitID        string    `json:"unitId,omitempty"`
	UnitCode      string    `json:"unitCode,omitempty"`
	Quantity      int       `json:"quantity"`
	Reason        string    `json:"reason"`
	PhotoURL      string    `json:"photoUrl"`
	AttachmentURL string    `json:"attachmentUrl"`
	Status        string    `json:"status"`
	Reviewer      string    `json:"reviewer"`
	ReviewComment string    `json:"reviewComment"`
	CreatedAt     time.Time `json:"createdAt"`
	ReviewedAt    time.Time `json:"reviewedAt,omitempty"`
	ProcessedAt   time.Time `json:"processedAt,omitempty"`
}

type MaterialDamageInput struct {
	MaterialID    string `json:"materialId"`
	ReporterID    string `json:"reporterId"`
	Reporter      string `json:"reporter"`
	UnitID        string `json:"unitId"`
	Quantity      int    `json:"quantity"`
	Reason        string `json:"reason"`
	PhotoURL      string `json:"photoUrl"`
	AttachmentURL string `json:"attachmentUrl"`
}

type FinancialAccount struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	UserName    string    `json:"userName"`
	Department  string    `json:"department"`
	GroupName   string    `json:"groupName"`
	Balance     float64   `json:"balance"`
	CreditLimit float64   `json:"creditLimit"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type FinancialAccountInput struct {
	UserID         string  `json:"userId"`
	UserName       string  `json:"userName"`
	GroupName      string  `json:"groupName"`
	CreditLimit    float64 `json:"creditLimit"`
	InitialBalance float64 `json:"initialBalance"`
	Actor          string  `json:"actor"`
}

type MaintenanceOrder struct {
	ID             string    `json:"id"`
	InstrumentID   string    `json:"instrumentId"`
	InstrumentName string    `json:"instrumentName"`
	Type           string    `json:"type"`
	Priority       string    `json:"priority"`
	Status         string    `json:"status"`
	Handler        string    `json:"handler"`
	Description    string    `json:"description"`
	Result         string    `json:"result"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MaintenanceInput struct {
	InstrumentID string    `json:"instrumentId"`
	Type         string    `json:"type"`
	Priority     string    `json:"priority"`
	Handler      string    `json:"handler"`
	Description  string    `json:"description"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	Actor        string    `json:"actor"`
}

type AnnouncementInput struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	Level       string `json:"level"`
	TargetScope string `json:"targetScope"`
	Target      string `json:"target"`
	UserID      string `json:"userId"`
	GroupName   string `json:"groupName"`
	Department  string `json:"department"`
	Actor       string `json:"actor"`
}

type LedgerAdjustmentInput struct {
	OriginalEntryID string  `json:"originalEntryId"`
	UserID          string  `json:"userId"`
	UserName        string  `json:"userName"`
	GroupName       string  `json:"groupName"`
	Amount          float64 `json:"amount"`
	Reason          string  `json:"reason"`
	Actor           string  `json:"actor"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	TargetType string    `json:"targetType"`
	TargetID   string    `json:"targetId"`
	OldValue   string    `json:"oldValue"`
	NewValue   string    `json:"newValue"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Operations struct {
	Dashboard          Dashboard        `json:"dashboard"`
	InUseInstruments   int              `json:"inUseInstruments"`
	AlertCount         int              `json:"alertCount"`
	UpdatedAt          time.Time        `json:"updatedAt"`
	ReservationTrend   []TrendPoint     `json:"reservationTrend"`
	InstrumentLoads    []InstrumentLoad `json:"instrumentLoads"`
	ApprovalEfficiency []ApprovalMetric `json:"approvalEfficiency"`
	Alerts             []OperationAlert `json:"alerts"`
}

type TrendPoint struct {
	Hour  string `json:"hour"`
	Count int    `json:"count"`
}

type InstrumentLoad struct {
	InstrumentName string  `json:"instrumentName"`
	Hours          float64 `json:"hours"`
}

type ApprovalMetric struct {
	Label string  `json:"label"`
	Hours float64 `json:"hours"`
}

type OperationAlert struct {
	Source string `json:"source"`
	Level  string `json:"level"`
	Body   string `json:"body"`
}
