import { cachedBusinessData } from "@/lib/business-data-cache";

export type Instrument = {
  id: string;
  name: string;
  category: string;
  department: string;
  groupName: string;
  status: string;
  location: string;
  hourlyRate: number;
  brand: string;
  model: string;
  assetCode: string;
  accessControlEnabled: boolean;
  accessControlGroup: string;
  accessControlPoint: string;
  description: string;
  technicalSpecs: string;
  bookingRule: string;
  maintenanceSummary: string;
  maxBookingHours: number;
  minAdvanceHours: number;
  cancelCutoffHours: number;
  checkinWindowMinutes: number;
  bookingWindowDays: number;
  bookingIntervalHours: number;
  serviceStartHour: number;
  serviceEndHour: number;
  usageCount: number;
};

export type TrainingCourse = {
  id: string;
  title: string;
  category: string;
  instrumentId?: string;
  instrumentName?: string;
  instructor: string;
  deliveryMode: string;
  durationHours: number;
  requiredForBooking: boolean;
  status: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingCoursePayload = {
  title: string;
  category: string;
  instrumentId?: string;
  instructor: string;
  deliveryMode: string;
  durationHours: number;
  requiredForBooking: boolean;
  status: string;
  description: string;
};

export type TrainingAuthorization = {
  id: string;
  userId?: string;
  userName: string;
  courseId?: string;
  courseTitle: string;
  instrumentId?: string;
  instrumentName?: string;
  status: string;
  expiresAt: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingAuthorizationPayload = {
  userId?: string;
  userName: string;
  courseId?: string;
  instrumentId?: string;
  status: string;
  expiresAt: string;
  notes: string;
};

export type TrainingQuestion = {
  id: string;
  title: string;
  questionType: string;
  options: string;
  correctAnswer: string;
  explanation: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingQuestionPayload = {
  title: string;
  questionType: string;
  options: string;
  correctAnswer: string;
  explanation: string;
  status: string;
};

export type TrainingExam = {
  id: string;
  userId?: string;
  userName: string;
  courseId?: string;
  courseTitle: string;
  score: number;
  passed: boolean;
  answers: string;
  status: string;
  notes: string;
  examAt: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingExamPayload = {
  userId?: string;
  userName: string;
  courseId?: string;
  score: number;
  passed: boolean;
  answers: string;
  status: string;
  notes: string;
  examAt: string;
};

export type TrainingPractical = {
  id: string;
  userId?: string;
  userName: string;
  instrumentId?: string;
  instrumentName?: string;
  assessor: string;
  score: number;
  result: string;
  notes: string;
  assessmentAt: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingPracticalPayload = {
  userId?: string;
  userName: string;
  instrumentId?: string;
  assessor: string;
  score: number;
  result: string;
  notes: string;
  assessmentAt: string;
};

export type TrainingRule = {
  id: string;
  instrumentId?: string;
  instrumentName?: string;
  requireTraining: boolean;
  requireExam: boolean;
  requireApproval: boolean;
  minScore: number;
  status: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type TrainingRulePayload = {
  instrumentId: string;
  requireTraining: boolean;
  requireExam: boolean;
  requireApproval: boolean;
  minScore: number;
  status: string;
  notes: string;
};

export type BusinessConfig = {
  id: string;
  tenantId?: string;
  domain: "workflow" | "billing";
  kind: string;
  title: string;
  category: string;
  scope: string;
  status: string;
  description: string;
  configJson: string;
  updatedBy: string;
  createdAt: string;
  updatedAt: string;
};

export type BusinessConfigPayload = {
  title: string;
  category: string;
  scope: string;
  status: string;
  description: string;
  configJson: string;
};

export type WorkflowConfigKind = "templates" | "rules" | "approvers" | "instances" | "exceptions";
export type BillingRuleKind = "instrument-rules" | "material-rules" | "invoices";

export type Space = {
  id: string;
  name: string;
  kind: string;
  department: string;
  location: string;
  capacity: number;
  status: string;
  accessControlPoint: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type SpacePayload = {
  name: string;
  kind: string;
  department: string;
  location: string;
  capacity: number;
  status: string;
  accessControlPoint: string;
  description: string;
};

export type SpaceReservation = {
  id: string;
  spaceId: string;
  spaceName: string;
  requesterId?: string;
  requester: string;
  purpose: string;
  startTime: string;
  endTime: string;
  status: string;
  createdAt: string;
};

export type SpaceReservationPayload = {
  spaceId: string;
  requesterId?: string;
  requester: string;
  purpose: string;
  startTime: string;
  endTime: string;
};

export type Sample = {
  id: string;
  code: string;
  name: string;
  ownerId?: string;
  ownerName: string;
  department: string;
  groupName: string;
  location: string;
  status: string;
  hazardLevel: string;
  storageCondition: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type SamplePayload = {
  code: string;
  name: string;
  ownerId?: string;
  ownerName: string;
  department: string;
  groupName: string;
  location: string;
  status: string;
  hazardLevel: string;
  storageCondition: string;
  description: string;
};

export type SampleMovement = {
  id: string;
  sampleId: string;
  sampleCode: string;
  sampleName: string;
  movementType: string;
  fromLocation: string;
  toLocation: string;
  reason: string;
  createdAt: string;
};

export type SampleMovementPayload = {
  sampleId: string;
  movementType: string;
  fromLocation: string;
  toLocation: string;
  reason: string;
};

export type LimsTask = {
  id: string;
  sampleId?: string;
  sampleCode?: string;
  instrumentId?: string;
  instrumentName?: string;
  title: string;
  assayType: string;
  priority: string;
  status: string;
  requesterId?: string;
  requesterName: string;
  dueAt: string;
  resultSummary: string;
  createdAt: string;
  updatedAt: string;
};

export type LimsTaskPayload = {
  sampleId?: string;
  instrumentId?: string;
  title: string;
  assayType: string;
  priority: string;
  status: string;
  requesterId?: string;
  requesterName: string;
  dueAt: string;
  resultSummary: string;
};

export type ElnRecord = {
  id: string;
  title: string;
  authorId?: string;
  authorName: string;
  project: string;
  linkedTaskId?: string;
  linkedTaskTitle?: string;
  content: string;
  status: string;
  signedAt: string;
  createdAt: string;
  updatedAt: string;
};

export type ElnRecordPayload = {
  title: string;
  authorId?: string;
  authorName: string;
  project: string;
  linkedTaskId?: string;
  content: string;
  status: string;
};

export type IotDevice = {
  id: string;
  name: string;
  vendor: string;
  deviceCode: string;
  instrumentId?: string;
  instrumentName?: string;
  online: boolean;
  status: string;
  lastSeenAt: string;
  telemetry: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type IotDevicePayload = {
  name: string;
  vendor: string;
  deviceCode: string;
  instrumentId?: string;
  online: boolean;
  status: string;
  telemetry: string;
  notes: string;
};

export type AssistantQuery = {
  id: string;
  question: string;
  answer: string;
  context: string;
  createdAt: string;
};

export type AssistantQueryPayload = {
  question: string;
  context: string;
};

export type Reservation = {
  id: string;
  userId?: string;
  instrumentId: string;
  instrumentName: string;
  userName: string;
  groupName: string;
  purpose: string;
  startTime: string;
  endTime: string;
  status: string;
  fee: number;
};

export type User = {
  id: string;
  tenantId: string;
  tenantName: string;
  tenantCode: string;
  name: string;
  email: string;
  phone: string;
  department: string;
  groupName: string;
  role: string;
  status: string;
  emailVerified: boolean;
  dingTalkUserId: string;
  dingTalkUnionId: string;
  dingTalkName: string;
  dingTalkBound: boolean;
  financeEnabled: boolean;
  authEpoch: number;
};

export type Tenant = {
  id: string;
  name: string;
  code: string;
  financeEnabled: boolean;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type TenantPayload = {
  name: string;
  code?: string;
  financeEnabled: boolean;
  status: string;
};

export type OrganizationUnit = {
  id: string;
  kind: "department" | "group";
  name: string;
  parentName: string;
  createdAt: string;
  updatedAt: string;
};

export type Notification = {
  id: string;
  userId?: string;
  groupName?: string;
  department?: string;
  targetScope: string;
  title: string;
  body: string;
  level: string;
  read: boolean;
  createdAt: string;
};

export type LedgerEntry = {
  id: string;
  userId?: string;
  userName?: string;
  reservationId?: string;
  groupName: string;
  description: string;
  amount: number;
  entryType: string;
  referenceId?: string;
  createdAt: string;
};

export type Dashboard = {
  todayReservations: number;
  pendingApprovals: number;
  inUseReservations: number;
  completedReservations: number;
  fulfillmentRate: number;
  activeInstruments: number;
  monthlyRevenue: number;
};

export type FooterSection = {
  title: string;
  lines: string[];
};

export type FooterSettings = {
  key: string;
  brandName: string;
  brandTagline: string;
  description: string;
  sections: FooterSection[];
  copyright: string;
  updatedBy: string;
  updatedAt: string;
};

export type FooterSettingsPayload = {
  brandName: string;
  brandTagline: string;
  description: string;
  sections: FooterSection[];
  copyright: string;
};

export type CopyEntry = {
  key: string;
  label: string;
  value: string;
  scope: string;
  description: string;
};

export type CopySettings = {
  key: string;
  entries: CopyEntry[];
  updatedBy: string;
  updatedAt: string;
};

export type CopySettingsPayload = {
  entries: CopyEntry[];
};

export type SMTPSettings = {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  fromEmail: string;
  fromName: string;
  passwordConfigured: boolean;
  updatedBy: string;
  updatedAt: string;
};

export type SMTPSettingsPayload = {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  password?: string;
  fromEmail: string;
  fromName: string;
};

export type WeChatSettings = {
  enabled: boolean;
  accountType: string;
  appId: string;
  serviceAccountName: string;
  templateId: string;
  token: string;
  encodingAesKey: string;
  appSecretConfigured: boolean;
  updatedBy: string;
  updatedAt: string;
};

export type WeChatSettingsPayload = {
  enabled: boolean;
  accountType: string;
  appId: string;
  appSecret?: string;
  serviceAccountName: string;
  templateId: string;
  token: string;
  encodingAesKey: string;
};

export type DingTalkSettings = {
  schemaVersion: number;
  enabled: boolean;
  clientId: string;
  clientSecret?: string;
  corpId: string;
  robotCode: string;
  oauthRedirectUri: string;
  eventCallbackUrl: string;
  eventAesKey?: string;
  eventToken?: string;
  clientSecretConfigured: boolean;
  eventAesKeyConfigured: boolean;
  eventTokenConfigured: boolean;
  updatedBy: string;
  updatedAt: string;
};

export type DingTalkSettingsPayload = {
  schemaVersion?: number;
  enabled: boolean;
  clientId: string;
  clientSecret?: string;
  corpId: string;
  robotCode: string;
  oauthRedirectUri: string;
  eventCallbackUrl: string;
  eventAesKey?: string;
  eventToken?: string;
};

export type DingTalkBinding = {
  bound: boolean;
  userId: string;
  unionId: string;
  name: string;
  authUrl?: string;
  state?: string;
  boundAt?: string;
  updatedAt?: string;
};

export type DingTalkBindingPayload = {
  authCode: string;
  state?: string;
};

export type NotificationChannelSettings = {
  smtp: SMTPSettings;
  wechat: WeChatSettings;
  dingtalk: DingTalkSettings;
};

export type AccessControlSettings = {
  enabled: boolean;
  vendor: "hikvision" | "dahua" | "custom";
  endpoint: string;
  clientId: string;
  accessGroup: string;
  autoGrantOnApproval: boolean;
  autoRevokeOnCompletion: boolean;
  clientSecretConfigured: boolean;
  updatedBy: string;
  updatedAt: string;
};

export type AccessControlSettingsPayload = {
  enabled: boolean;
  vendor: "hikvision" | "dahua" | "custom";
  endpoint: string;
  clientId: string;
  clientSecret?: string;
  accessGroup: string;
  autoGrantOnApproval: boolean;
  autoRevokeOnCompletion: boolean;
};

export type InstrumentFilters = {
  search?: string;
  category?: string;
  department?: string;
  group?: string;
  status?: string;
  limit?: number;
  offset?: number;
};

export type RegisterPayload = {
  tenantId: string;
  tenantCode?: string;
  accountType: "user";
  name: string;
  phone: string;
  email: string;
  password: string;
  department: string;
  verificationCode: string;
};

export type EmailVerificationCodePayload = {
  tenantId: string;
  tenantCode?: string;
  email: string;
};

export type EmailVerificationCodeResponse = {
  sent: boolean;
  message: string;
};

export type ReservationPayload = {
  instrumentId: string;
  userName?: string;
  purpose: string;
  startTime: string;
  endTime: string;
  idempotencyKey?: string;
};

export type ReservationPeriodPayload = {
  startTime: string;
  endTime: string;
};

export type ReservationBatchPayload = {
  instrumentId: string;
  userName?: string;
  purpose: string;
  periods: ReservationPeriodPayload[];
  idempotencyKey?: string;
};

export type InstrumentPayload = Omit<Instrument, "id" | "usageCount">;

export type UserReviewPayload = {
  tenantId?: string;
  role: string;
  groupName: string;
  department: string;
  email: string;
  phone: string;
  status: string;
  actor?: string;
};

export type OrganizationUnitPayload = {
  kind: "department" | "group";
  name: string;
  parentName?: string;
};

export type Slot = {
  startTime: string;
  endTime: string;
  status: string;
  reason: string;
};

export type Material = {
  id: string;
  name: string;
  productType: string;
  category: string;
  subcategory: string;
  spec: string;
  unit: string;
  unitPrice: number;
  stock: number;
  warningLine: number;
  supplier: string;
  manufacturer: string;
  batchNo: string;
  catalogNo: string;
  casNo: string;
  grade: string;
  concentration: string;
  parentMaterialId: string;
  parentMaterialName: string;
  dilutionFactor: string;
  preparationMethod: string;
  storageCondition: string;
  storageRoom: string;
  storageCabinet: string;
  storageLayer: string;
  storageSlot: string;
  tenderContract: string;
  contractNo: string;
  certificateUrl: string;
  standardCertificateUrl: string;
  attachmentUrl: string;
  qrCode: string;
  expiresAt: string;
  openedAt: string;
  openExpireDays: number;
  openExpiresAt: string;
  freezeThawCount: number;
  freezeThawLimit: number;
  approvalRequired: boolean;
  nearExpiryDays: number;
  damagedQuantity: number;
  batches: MaterialBatch[];
  units: MaterialUnit[];
  status: string;
};

export type MaterialBatch = {
  id: string;
  materialId: string;
  batchNo: string;
  quantity: number;
  expiresAt: string;
  location: string;
  units: MaterialUnit[];
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type MaterialUnit = {
  id: string;
  materialId: string;
  batchId?: string;
  batchNo?: string;
  unitCode: string;
  expiresAt: string;
  location: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type MaterialPayload = Omit<Material, "id" | "parentMaterialName" | "openExpiresAt" | "damagedQuantity" | "batches" | "units">;

export type MaterialImportResult = {
  created: number;
  updated: number;
  skipped: number;
  errors: string[];
};

export type MaterialCategory = {
  id: string;
  name: string;
  parentName: string;
  displayOrder: number;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type MaterialCategoryPayload = {
  name: string;
  parentName?: string;
  displayOrder?: number;
  status: string;
};

export type MaterialAlertAction = {
  id: string;
  materialId: string;
  materialName: string;
  alertType: string;
  action: string;
  comment: string;
  actor: string;
  createdAt: string;
};

export type MaterialAlertActionPayload = {
  alertType: string;
  action: "handled" | "ignored";
  comment?: string;
};

export type MaterialAnalytics = {
  productTotal: number;
  stockTotal: number;
  standardTotal: number;
  todayUsageTotal: number;
  nearExpiryTotal: number;
  expiredTotal: number;
  lowStockTotal: number;
  damagedTotal: number;
  monthlyConsumption: { month: string; quantity: number }[];
  topConsumedMaterials: { materialId: string; materialName: string; quantity: number }[];
  damageByReason: { reason: string; quantity: number }[];
  productTypeBreakdown: { label: string; count: number; stock: number }[];
  categoryBreakdown: { label: string; count: number; stock: number }[];
  latestAlertActions: MaterialAlertAction[];
};

export type UserMembershipPayload = {
  tenantId: string;
  role: string;
  groupName: string;
  department: string;
  status: string;
};

export type StockAdjustmentPayload = {
  changeQty: number;
  reason: string;
  batchId?: string;
  batchNo?: string;
  unitId?: string;
  expiresAt?: string;
  location?: string;
};

export type MaterialRequest = {
  id: string;
  materialId: string;
  materialName: string;
  requesterId?: string;
  requester: string;
  groupName: string;
  batchId?: string;
  batchNo?: string;
  unitId?: string;
  unitCode?: string;
  quantity: number;
  purpose: string;
  status: string;
  createdAt: string;
};

export type MaterialPurchase = {
  id: string;
  materialId: string;
  materialName: string;
  requesterId?: string;
  requester: string;
  groupName: string;
  quantity: number;
  estimatedUnitPrice: number;
  supplier: string;
  reason: string;
  status: string;
  createdAt: string;
};

export type InventoryLedgerEntry = {
  id: string;
  materialId: string;
  materialName: string;
  requestId?: string;
  purchaseId?: string;
  damageId?: string;
  changeQty: number;
  reason: string;
  createdAt: string;
};

export type MaterialRequestPayload = {
  materialId: string;
  requester?: string;
  batchId?: string;
  unitId: string;
  quantity: 1;
  purpose: string;
};

export type MaterialPurchasePayload = {
  materialId: string;
  requester?: string;
  quantity: number;
  estimatedUnitPrice: number;
  supplier: string;
  reason: string;
};

export type MaterialDamage = {
  id: string;
  materialId: string;
  materialName: string;
  reporterId?: string;
  reporter: string;
  groupName: string;
  batchId?: string;
  batchNo?: string;
  unitId?: string;
  unitCode?: string;
  quantity: number;
  reason: string;
  photoUrl: string;
  attachmentUrl: string;
  status: string;
  reviewer: string;
  reviewComment: string;
  createdAt: string;
  reviewedAt?: string;
  processedAt?: string;
};

export type MaterialDamagePayload = {
  materialId: string;
  unitId: string;
  quantity: 1;
  reason: string;
  photoUrl?: string;
  attachmentUrl?: string;
};

export type MaintenanceOrder = {
  id: string;
  instrumentId: string;
  instrumentName: string;
  type: string;
  priority: string;
  status: string;
  handler: string;
  description: string;
  result: string;
  startTime: string;
  endTime: string;
  createdAt: string;
};

export type MaintenancePayload = {
  instrumentId: string;
  type: string;
  priority: string;
  handler: string;
  description: string;
  startTime: string;
  endTime: string;
};

export type AuditEvent = {
  id: string;
  actor: string;
  action: string;
  targetType: string;
  targetId: string;
  oldValue: string;
  newValue: string;
  createdAt: string;
};

export type Operations = {
  dashboard: Dashboard;
  inUseInstruments: number;
  alertCount: number;
  updatedAt: string;
  reservationTrend: { hour: string; count: number }[];
  instrumentLoads: { instrumentName: string; hours: number }[];
  approvalEfficiency: { label: string; hours: number }[];
  alerts: { source: string; level: string; body: string }[];
};

export type FinancialAccount = {
  id: string;
  userId: string;
  userName: string;
  department: string;
  groupName: string;
  balance: number;
  creditLimit: number;
  updatedAt: string;
};

export type FinancialAccountPayload = {
  userId: string;
  creditLimit: number;
  initialBalance?: number;
};

export type LoginPayload = {
  tenantId?: string;
  tenantCode?: string;
  email: string;
  password: string;
  device?: string;
};

export type UserProfilePayload = {
  name?: string;
  phone?: string;
  department?: string;
  groupName?: string;
};

export type PasswordChangePayload = {
  currentPassword: string;
  newPassword: string;
};

export type AuthResponse = {
  token: string;
  expiresAt: string;
  user: User;
};

const serverBaseUrl = process.env.API_BASE_URL ?? "http://localhost:8090";
const browserBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || "";
const authTokenKey = "lirs.authToken";
const publicSettingsRevalidateSeconds = 60;
const currentUserCacheSeconds = 60;

type ServerRequestOptions = {
  businessCacheKey?: string;
  cacheTtlSeconds?: number;
  cache?: RequestCache;
  revalidate?: number;
  skipAuth?: boolean;
};

async function request<T>(path: string, init?: RequestInit, options: ServerRequestOptions = {}): Promise<T> {
  const authHeaders = options.skipAuth ? {} : await serverAuthHeaders();
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  for (const [key, value] of Object.entries(authHeaders)) {
    headers.set(key, value);
  }
  const fetchOptions: RequestInit & { next?: { revalidate?: number | false } } = {
    ...init,
    headers,
  };
  if (options.revalidate !== undefined) {
    fetchOptions.next = { revalidate: options.revalidate };
  } else {
    fetchOptions.cache = options.cache ?? "no-store";
  }
  const load = async () => {
    const res = await fetch(`${serverBaseUrl}${path}`, fetchOptions);
    if (!res.ok) {
      if (res.status === 401 && typeof window === "undefined") {
        const { redirect } = await import("next/navigation");
        redirect("/login");
      }
      throw new Error(`API ${path} failed with ${res.status}`);
    }
    return (await res.json()) as T;
  };
  if (options.businessCacheKey) {
    return cachedBusinessData(createBusinessDataCacheKey(path, init, authHeaders, options.businessCacheKey), load, options.cacheTtlSeconds);
  }
  return load();
}

function businessRequest<T>(path: string, tag: string, init?: RequestInit): Promise<T> {
  return request<T>(path, init, { businessCacheKey: tag });
}

function createBusinessDataCacheKey(path: string, init: RequestInit | undefined, authHeaders: Record<string, string>, tag: string) {
  return JSON.stringify([tag, init?.method ?? "GET", path, authHeaders.Authorization ?? ""]);
}

export const api = {
  dashboard: () => businessRequest<Dashboard>("/api/dashboard", "dashboard"),
  tenants: () => businessRequest<Tenant[]>("/api/tenants", "tenants"),
  footerSettings: () => request<FooterSettings>("/api/footer-settings"),
  copySettings: () => request<CopySettings>("/api/copy-settings"),
  cachedFooterSettings: () => request<FooterSettings>("/api/footer-settings", undefined, { revalidate: publicSettingsRevalidateSeconds, skipAuth: true }),
  cachedCopySettings: () => request<CopySettings>("/api/copy-settings", undefined, { revalidate: publicSettingsRevalidateSeconds, skipAuth: true }),
  notificationChannelSettings: () => businessRequest<NotificationChannelSettings>("/api/notification-channel-settings", "notification-channel-settings"),
  dingTalkSettings: (tenantId?: string) => businessRequest<DingTalkSettings>(withQuery("/api/notification-channel-settings/dingtalk", { tenantId }), `dingtalk-settings:${tenantId ?? ""}`),
  dingTalkBinding: () => request<DingTalkBinding>("/api/me/dingtalk-binding"),
  accessControlSettings: () => businessRequest<AccessControlSettings>("/api/access-control-settings", "access-control-settings"),
  instruments: (filters: InstrumentFilters = {}) => businessRequest<Instrument[]>(withQuery("/api/instruments", filters), "instruments"),
  instrument: (id: string) => businessRequest<Instrument>(`/api/instruments/${id}`, "instruments"),
  slots: (id: string, days = 7) => businessRequest<Slot[]>(`/api/instruments/${id}/slots?days=${days}`, "instrument-slots"),
  trainingCourses: () => businessRequest<TrainingCourse[]>("/api/training/courses", "training-courses"),
  trainingAuthorizations: () => businessRequest<TrainingAuthorization[]>("/api/training/authorizations", "training-authorizations"),
  trainingQuestions: () => businessRequest<TrainingQuestion[]>("/api/training/questions", "training-questions"),
  trainingExams: () => businessRequest<TrainingExam[]>("/api/training/exams", "training-exams"),
  trainingPracticals: () => businessRequest<TrainingPractical[]>("/api/training/practicals", "training-practicals"),
  trainingRules: () => businessRequest<TrainingRule[]>("/api/training/rules", "training-rules"),
  workflowConfigs: (kind: WorkflowConfigKind) => businessRequest<BusinessConfig[]>(`/api/workflows/${kind}`, "workflow-configs"),
  billingRules: (kind: BillingRuleKind) => businessRequest<BusinessConfig[]>(`/api/billing/${kind}`, "billing-rules"),
  spaces: () => businessRequest<Space[]>("/api/spaces", "spaces"),
  spaceReservations: () => businessRequest<SpaceReservation[]>("/api/space-reservations", "space-reservations"),
  samples: () => businessRequest<Sample[]>("/api/samples", "samples"),
  sampleMovements: () => businessRequest<SampleMovement[]>("/api/sample-movements", "sample-movements"),
  limsTasks: () => businessRequest<LimsTask[]>("/api/lims/tasks", "lims-tasks"),
  elnRecords: () => businessRequest<ElnRecord[]>("/api/eln/records", "eln-records"),
  iotDevices: () => businessRequest<IotDevice[]>("/api/iot/devices", "iot-devices"),
  assistantQueries: () => businessRequest<AssistantQuery[]>("/api/ai-assistant", "assistant-queries"),
  reservations: () => businessRequest<Reservation[]>("/api/reservations", "reservations"),
  users: () => businessRequest<User[]>("/api/users", "users"),
  notifications: () => businessRequest<Notification[]>("/api/notifications", "notifications"),
  ledger: () => businessRequest<LedgerEntry[]>("/api/ledger", "ledger"),
  financialAccounts: () => businessRequest<FinancialAccount[]>("/api/financial-accounts", "financial-accounts"),
  materials: () => businessRequest<Material[]>("/api/materials", "materials"),
  materialAnalytics: () => businessRequest<MaterialAnalytics>("/api/materials/analytics", "material-analytics"),
  materialCategories: () => businessRequest<MaterialCategory[]>("/api/materials/categories", "material-categories"),
  materialAlertActions: () => businessRequest<MaterialAlertAction[]>("/api/materials/alert-actions", "material-alert-actions"),
  materialByQRCode: (code: string) => businessRequest<Material>(`/api/materials/scan/${encodeURIComponent(code)}`, "materials"),
  inventoryLedger: () => businessRequest<InventoryLedgerEntry[]>("/api/inventory-ledger", "inventory-ledger"),
  materialRequests: () => businessRequest<MaterialRequest[]>("/api/material-requests", "material-requests"),
  materialPurchases: () => businessRequest<MaterialPurchase[]>("/api/material-purchases", "material-purchases"),
  materialDamages: () => businessRequest<MaterialDamage[]>("/api/material-damages", "material-damages"),
  maintenance: () => businessRequest<MaintenanceOrder[]>("/api/maintenance", "maintenance"),
  auditEvents: () => businessRequest<AuditEvent[]>("/api/audit-events", "audit-events"),
  operations: () => businessRequest<Operations>("/api/operations", "operations"),
  me: () => request<User>("/api/me", undefined, { businessCacheKey: "current-user-required", cacheTtlSeconds: currentUserCacheSeconds }),
  currentUserOptional: () => requestOptional<User>("/api/me", undefined, { businessCacheKey: "current-user-optional", cacheTtlSeconds: currentUserCacheSeconds }),
  organizationUnits: (kind?: "department" | "group", tenantId?: string) => businessRequest<OrganizationUnit[]>(withQuery("/api/organization-units", { kind, tenantId }), "organization-units"),
};

export function createDefaultFooterSettings(): FooterSettings {
  return {
    key: "footer",
    brandName: "LIRS 2026 实验室运营系统",
    brandTagline: "仪器预约、审批、使用、耗材、财务与审计闭环平台",
    description:
      "系统数据统一写入 PostgreSQL，登录会话、审批、库存、财务流水和审计记录均从数据库读取；Redis 用于缓存与事件队列。",
    sections: [
      {
        title: "技术栈",
        lines: [
          "TypeScript / Next.js / React / Tailwind CSS / shadcn/ui / Lucide Icons",
          "Go / Gin / Hono / Zod / Drizzle ORM / PostgreSQL 15+ / Redis 7+",
        ],
      },
      {
        title: "运行信息",
        lines: ["Hono API Gateway: 8090", "Go Core API: 8081"],
      },
    ],
    copyright: "© 2026 LIRS. All rights reserved.",
    updatedBy: "system",
    updatedAt: "",
  };
}

export function createDefaultCopySettings(): CopySettings {
  return {
    key: "copy",
    entries: uniqueCopyEntries([
      copyEntry("首页", "主导航", "首页", "nav", "顶部首页入口"),
      copyEntry("仪器预约", "主导航", "仪器预约", "nav", "顶部仪器预约入口"),
      copyEntry("资源", "主导航", "资源", "nav", "顶部资源分组"),
      copyEntry("业务", "主导航", "业务", "nav", "顶部业务分组"),
      copyEntry("培训", "主导航", "培训", "nav", "顶部培训分组"),
      copyEntry("更多", "主导航", "更多", "nav", "顶部更多分组"),
      copyEntry("管理中心", "主导航", "管理中心", "nav", "顶部管理中心入口"),
      copyEntry("管理后台", "主导航", "管理后台", "nav", "移动端管理入口"),
      copyEntry("登录", "按钮", "登录", "button", "登录入口按钮"),
      copyEntry("注册", "按钮", "注册", "button", "注册入口按钮"),
      copyEntry("退出登录", "按钮", "退出登录", "button", "退出当前会话"),
      copyEntry("退出所有设备", "按钮", "退出所有设备", "button", "退出全部会话"),
      copyEntry("搜索", "按钮", "搜索", "button", "搜索按钮"),
      copyEntry("筛选", "按钮", "筛选", "button", "筛选按钮"),
      copyEntry("保存", "按钮", "保存", "button", "保存按钮"),
      copyEntry("修改", "按钮", "修改", "button", "修改按钮"),
      copyEntry("删除", "按钮", "删除", "button", "删除按钮"),
      copyEntry("取消", "按钮", "取消", "button", "取消按钮"),
      copyEntry("通过", "按钮", "通过", "button", "通过按钮"),
      copyEntry("拒绝", "按钮", "拒绝", "button", "拒绝按钮"),
      copyEntry("提交", "按钮", "提交", "button", "提交按钮"),
      copyEntry("详情", "按钮", "详情", "button", "详情按钮"),
      copyEntry("查看详情", "按钮", "查看详情", "button", "查看详情按钮"),
      copyEntry("新建申购", "按钮", "新建申购", "button", "新建申购按钮"),
      copyEntry("新建预约", "按钮", "新建预约", "button", "新建预约按钮"),
      copyEntry("提交预约", "按钮", "提交预约", "button", "提交预约按钮"),
      copyEntry("提交申领", "按钮", "提交申领", "button", "提交申领按钮"),
      copyEntry("提交申购", "按钮", "提交申购", "button", "提交申购按钮"),
      copyEntry("确认通过", "按钮", "确认通过", "button", "通过确认按钮"),
      copyEntry("确认拒绝", "按钮", "确认拒绝", "button", "拒绝确认按钮"),
      copyEntry("确认取消", "按钮", "确认取消", "button", "取消确认按钮"),
      copyEntry("导出流水", "按钮", "导出流水", "button", "导出流水按钮"),
      copyEntry("导出 CSV", "按钮", "导出 CSV", "button", "导出 CSV 按钮"),
      copyEntry("标记下单", "按钮", "标记下单", "button", "申购下单按钮"),
      copyEntry("到货入库", "按钮", "到货入库", "button", "申购入库按钮"),
      copyEntry("出库", "按钮", "出库", "button", "申领出库按钮"),
      copyEntry("签到", "按钮", "签到", "button", "预约签到按钮"),
      copyEntry("签退并入账", "按钮", "签退并入账", "button", "预约签退按钮"),
      copyEntry("返回系统首页", "按钮", "返回系统首页", "button", "返回首页按钮"),
      copyEntry("返回上一页", "按钮", "返回上一页", "button", "返回上一页按钮"),
      copyEntry("仪器预约大厅", "页面标题", "仪器预约大厅", "page", "首页主标题"),
      copyEntry("通知中心", "页面标题", "通知中心", "page", "通知中心标题"),
      copyEntry("财务管理", "页面标题", "财务管理", "page", "财务页标题"),
      copyEntry("平台配置中心", "页面标题", "平台配置中心", "page", "后台配置页标题"),
      copyEntry("系统基础配置", "页面标题", "系统基础配置", "page", "Footer 页面配置标题"),
      copyEntry("实验室运营系统", "品牌", "实验室运营系统", "brand", "系统名称"),
      copyEntry("实验室运营管理", "品牌", "实验室运营管理", "brand", "后台管理标题"),
      copyEntry("主导航", "辅助", "主导航", "meta", "主导航分组标题"),
      copyEntry("移动端主导航", "辅助", "移动端主导航", "meta", "移动端导航无障碍标题"),
      copyEntry("当前账号", "辅助", "当前账号", "meta", "个人菜单账号区标题"),
      copyEntry("通知", "辅助", "通知", "meta", "顶部通知入口标题"),
      copyEntry("主题", "辅助", "主题", "meta", "暗色模式切换按钮标题"),
      copyEntry("仪器分类", "筛选", "仪器分类", "filter", "顶部搜索分类选择标题"),
      copyEntry("全部分类", "筛选", "全部分类", "filter", "顶部搜索全部分类选项"),
      copyEntry("快速查找仪器...", "占位符", "快速查找仪器...", "placeholder", "顶部搜索框"),
      copyEntry("搜索仪器名称、型号、部门...", "占位符", "搜索仪器名称、型号、部门...", "placeholder", "首页筛选"),
      copyEntry("搜索设备、厂商、编码、仪器...", "占位符", "搜索设备、厂商、编码、仪器...", "placeholder", "IoT 搜索"),
      copyEntry("搜索问题、回答或背景...", "占位符", "搜索问题、回答或背景...", "placeholder", "AI 助手搜索"),
      copyEntry("搜索申请人、仪器、团队...", "占位符", "搜索申请人、仪器、团队...", "placeholder", "审批搜索"),
      copyEntry("搜索产品、申请人、用途", "占位符", "搜索产品、申请人、用途", "placeholder", "产品申领搜索"),
      copyEntry("搜索课程、讲师、仪器...", "占位符", "搜索课程、讲师、仪器...", "placeholder", "培训课程搜索"),
      copyEntry("搜索标题、作者、项目、任务...", "占位符", "搜索标题、作者、项目、任务...", "placeholder", "ELN 搜索"),
      copyEntry("搜索编号、名称、负责人...", "占位符", "搜索编号、名称、负责人...", "placeholder", "样本搜索"),
      copyEntry("普通用户入口", "首页分组", "普通用户入口", "group", "游客首页普通用户入口"),
      copyEntry("个人工作台", "首页分组", "个人工作台", "group", "普通用户首页工作台"),
      copyEntry("实验资源中心", "首页分组", "实验资源中心", "group", "实验资源入口"),
      copyEntry("业务流程中心", "首页分组", "业务流程中心", "group", "业务入口"),
      copyEntry("培训与准入中心", "首页分组", "培训与准入中心", "group", "培训入口"),
      copyEntry("空间资源中心", "首页分组", "空间资源中心", "group", "空间入口"),
      copyEntry("扩展能力中心", "首页分组", "扩展能力中心", "group", "扩展系统入口"),
      copyEntry("财务与计费中心", "首页分组", "财务与计费中心", "group", "财务入口"),
      copyEntry("管理员工作台", "首页分组", "管理员工作台", "group", "管理员首页入口"),
      copyEntry("资源与准入管理", "首页分组", "资源与准入管理", "group", "管理员资源管理入口"),
      copyEntry("组织与配置", "首页分组", "组织与配置", "group", "管理员组织配置入口"),
      copyEntry("前台常用入口", "首页分组", "前台常用入口", "group", "管理员前台快捷入口"),
      copyEntry("仪器资源管理", "首页模块", "仪器资源管理", "module", "首页仪器资源卡片标题"),
      copyEntry("资源目录", "首页模块", "资源目录", "module", "首页资源目录卡片标题"),
      copyEntry("申领管理", "首页模块", "申领管理", "module", "首页申领管理卡片标题"),
      copyEntry("资源申购", "首页模块", "资源申购", "module", "首页资源申购卡片标题"),
      copyEntry("消息中心", "首页模块", "消息中心", "module", "首页消息中心卡片标题"),
      copyEntry("个人信息", "首页模块", "个人信息", "module", "首页个人信息卡片标题"),
      copyEntry("账户设置", "首页模块", "账户设置", "module", "首页账户设置卡片标题"),
      copyEntry("培训与准入总览", "首页模块", "培训与准入总览", "module", "首页培训总览卡片标题"),
      copyEntry("课程管理", "首页模块", "课程管理", "module", "首页课程管理卡片标题"),
      copyEntry("授权记录", "首页模块", "授权记录", "module", "首页授权记录卡片标题"),
      copyEntry("在线考试", "首页模块", "在线考试", "module", "首页在线考试卡片标题"),
      copyEntry("空间资源", "首页模块", "空间资源", "module", "首页空间资源卡片标题"),
      copyEntry("LIMS 检测任务", "首页模块", "LIMS 检测任务", "module", "首页 LIMS 卡片标题"),
      copyEntry("ELN 实验记录", "首页模块", "ELN 实验记录", "module", "首页 ELN 卡片标题"),
      copyEntry("样本管理", "首页模块", "样本管理", "module", "首页样本卡片标题"),
      copyEntry("IoT 设备中心", "首页模块", "IoT 设备中心", "module", "首页 IoT 卡片标题"),
      copyEntry("AI 助手", "首页模块", "AI 助手", "module", "首页 AI 卡片标题"),
      copyEntry("数据中台", "首页模块", "数据中台", "module", "首页数据中台卡片标题"),
      copyEntry("我的申请", "首页模块", "我的申请", "module", "首页工作台卡片标题"),
      copyEntry("预约记录", "首页模块", "预约记录", "module", "首页预约卡片标题"),
      copyEntry("审批中心", "首页模块", "审批中心", "module", "首页审批卡片标题"),
      copyEntry("工作概览", "首页模块", "工作概览", "module", "首页管理工作台卡片标题"),
      copyEntry("运营看板", "首页模块", "运营看板", "module", "首页运营看板卡片标题"),
      copyEntry("运营分析中心", "首页模块", "运营分析中心", "module", "首页运营分析卡片标题"),
      copyEntry("通知管理", "首页模块", "通知管理", "module", "首页通知管理卡片标题"),
      copyEntry("安全审计与合规", "首页模块", "安全审计与合规", "module", "首页安全审计卡片标题"),
      copyEntry("仪器资源后台", "首页模块", "仪器资源后台", "module", "首页仪器后台卡片标题"),
      copyEntry("资源管理后台", "首页模块", "资源管理后台", "module", "首页资源后台卡片标题"),
      copyEntry("工单与设备维护", "首页模块", "工单与设备维护", "module", "首页维护卡片标题"),
      copyEntry("用户管理", "首页模块", "用户管理", "module", "首页用户管理卡片标题"),
      copyEntry("平台配置中心", "首页模块", "平台配置中心", "module", "首页平台配置卡片标题"),
      copyEntry("组织架构管理", "首页模块", "组织架构管理", "module", "首页组织架构卡片标题"),
      copyEntry("租户配置", "首页模块", "租户配置", "module", "首页租户卡片标题"),
      copyEntry("财务模块开关", "首页模块", "财务模块开关", "module", "首页财务开关卡片标题"),
      copyEntry("通知通道配置", "首页模块", "通知通道配置", "module", "首页通知通道卡片标题"),
      copyEntry("第三方集成", "首页模块", "第三方集成", "module", "首页第三方集成卡片标题"),
      copyEntry("系统基础配置", "首页模块", "系统基础配置", "module", "首页 footer 配置卡片标题"),
      copyEntry("文案中心", "首页模块", "文案中心", "module", "首页文案中心卡片标题"),
    ]),
    updatedBy: "system",
    updatedAt: "",
  };
}

export function createDefaultAccessControlSettings(): AccessControlSettings {
  return {
    enabled: false,
    vendor: "hikvision",
    endpoint: "",
    clientId: "",
    accessGroup: "",
    autoGrantOnApproval: true,
    autoRevokeOnCompletion: true,
    clientSecretConfigured: false,
    updatedBy: "system",
    updatedAt: "",
  };
}

export function createDefaultDingTalkSettings(): DingTalkSettings {
  return {
    schemaVersion: 2,
    enabled: false,
    clientId: "",
    clientSecret: "",
    corpId: "",
    robotCode: "",
    oauthRedirectUri: "",
    eventCallbackUrl: "",
    eventAesKey: "",
    eventToken: "",
    clientSecretConfigured: false,
    eventAesKeyConfigured: false,
    eventTokenConfigured: false,
    updatedBy: "system",
    updatedAt: "",
  };
}

function copyEntry(key: string, label: string, value: string, scope: string, description: string): CopyEntry {
  return { key, label, value, scope, description };
}

export function copyText(settings: CopySettings | null | undefined, key: string, fallback = key) {
  const value = settings?.entries.find((entry) => entry.key === key)?.value?.trim();
  return value || fallback;
}

function uniqueCopyEntries(entries: CopyEntry[]) {
  const seen = new Set<string>();
  const result: CopyEntry[] = [];
  for (const entry of entries) {
    if (!entry.key || seen.has(entry.key)) {
      continue;
    }
    seen.add(entry.key);
    result.push(entry);
  }
  return result;
}

export function getStoredToken() {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(authTokenKey) ?? "";
}

export function storeAuth(_auth: AuthResponse) {
  // Auth now lives in the HttpOnly cookie set by /api/login.
}

export function clearAuth() {
  window.localStorage.removeItem(authTokenKey);
  // Clear legacy non-HttpOnly cookies from older builds. The current auth cookie is cleared by /api/logout.
  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie = `${authTokenKey}=; Path=/; SameSite=Lax${secure}; Max-Age=0`;
}

export async function browserLogin(payload: LoginPayload): Promise<AuthResponse> {
  const auth = await browserRequestWithOptions<AuthResponse>("/api/login", "POST", payload, { skipAuth: true });
  if (typeof window !== "undefined") {
    window.localStorage.removeItem(authTokenKey);
  }
  return auth;
}

export async function browserMe(): Promise<User> {
  return browserRequest<User>("/api/me", "GET");
}

export async function browserLogout(): Promise<void> {
  try {
    await browserRequest<{ ok: boolean }>("/api/logout", "POST");
  } finally {
    clearAuth();
  }
}

export async function browserLogoutAll(): Promise<void> {
  try {
    await browserRequest<{ ok: boolean }>("/api/logout-all", "POST");
  } finally {
    clearAuth();
  }
}

export async function browserRequest<T>(path: string, method: string, payload?: unknown): Promise<T> {
  return browserRequestWithOptions<T>(path, method, payload);
}

async function browserRequestWithOptions<T>(
  path: string,
  method: string,
  payload?: unknown,
  options: { skipAuth?: boolean } = {},
): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const res = await fetch(`${browserBaseUrl}${path}`, {
    method,
    headers,
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `请求失败: ${res.status}`);
  }
  return (await res.json()) as T;
}

async function requestOptional<T>(path: string, init?: RequestInit, options: ServerRequestOptions = {}): Promise<T | null> {
  const authHeaders = await serverAuthHeaders();
  if (typeof window === "undefined" && !authHeaders.Authorization) {
    return null;
  }
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  for (const [key, value] of Object.entries(authHeaders)) {
    headers.set(key, value);
  }
  const load = async () => {
    const res = await fetch(`${serverBaseUrl}${path}`, {
      ...init,
      headers,
      cache: options.cache ?? "no-store",
    });
    if (res.status === 401) {
      return null;
    }
    if (!res.ok) {
      throw new Error(`API ${path} failed with ${res.status}`);
    }
    return (await res.json()) as T;
  };
  if (options.businessCacheKey) {
    return cachedBusinessData(createBusinessDataCacheKey(path, init, authHeaders, options.businessCacheKey), load, options.cacheTtlSeconds);
  }
  return load();
}

async function serverAuthHeaders(): Promise<Record<string, string>> {
  if (typeof window !== "undefined") {
    return {};
  }
  try {
    const { cookies } = await import("next/headers");
    const token = (await cookies()).get(authTokenKey)?.value ?? "";
    return token ? { Authorization: `Bearer ${token}` } : {};
  } catch {
    return {};
  }
}

export async function browserPost<T>(path: string, payload: unknown): Promise<T> {
  return browserRequest<T>(path, "POST", payload);
}

export async function browserPatch<T>(path: string, payload?: unknown): Promise<T> {
  return browserRequest<T>(path, "PATCH", payload);
}

export async function browserDelete<T>(path: string): Promise<T> {
  return browserRequest<T>(path, "DELETE");
}

function withQuery(path: string, params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && String(value).trim() !== "") {
      query.set(key, String(value));
    }
  }
  const suffix = query.toString();
  return suffix ? `${path}?${suffix}` : path;
}
