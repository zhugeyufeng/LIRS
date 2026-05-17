import Link from "next/link";
import {
  Bot,
  Bell,
  BarChart3,
  CalendarCheck2,
  ClipboardCheck,
  Cpu,
  Database,
  FileText,
  FlaskConical,
  LayoutDashboard,
  MonitorPlay,
  Building2,
  PackageCheck,
  PackageSearch,
  GraduationCap,
  Settings2,
  ShieldCheck,
  TestTube2,
  ShoppingCart,
  ThermometerSun,
  Type,
  UserRound,
  UsersRound,
  Wallet,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import { AppShell } from "@/components/app-shell";
import { api, copyText, createDefaultCopySettings } from "@/lib/api";
import { canAccessAdminSection, isAnyAdminRole, isTenantAdminRole, type AdminSection } from "@/lib/permissions";

type ModuleEntry = {
  href: string;
  title: string;
  description: string;
  icon: LucideIcon;
  adminSection?: AdminSection;
};

type ModuleGroup = {
  title: string;
  description: string;
  modules: ModuleEntry[];
};

type CopyTranslator = (key: string, fallback?: string) => string;

const resourceModules: ModuleEntry[] = [
  {
    href: "/instruments",
    title: "仪器资源管理",
    description: "查看仪器列表、部门分类、详情和在线预约入口。",
    icon: CalendarCheck2,
  },
  {
    href: "/materials",
    title: "资源目录",
    description: "分区查看标准品、试剂和耗材目录、唯一编号、库存状态和预警信息。",
    icon: PackageSearch,
  },
  {
    href: "/materials/standards",
    title: "标准品目录",
    description: "按批次和唯一编号查看标准品，申领时选择具体编号。",
    icon: TestTube2,
  },
  {
    href: "/materials/reagents",
    title: "试剂目录",
    description: "独立查看试剂目录、批次、库位、有效期和库存预警。",
    icon: FlaskConical,
  },
  {
    href: "/materials/consumables",
    title: "耗材目录",
    description: "独立查看耗材最小单位编号、库位和可申领库存。",
    icon: PackageCheck,
  },
  {
    href: "/materials/requests",
    title: "申领管理",
    description: "查看资源申领记录，跟踪审批和出库状态。",
    icon: PackageCheck,
  },
  {
    href: "/materials/purchases",
    title: "资源申购",
    description: "提交采购申请，跟踪审批、下单和入库。",
    icon: ShoppingCart,
  },
];

const accessModules: ModuleEntry[] = [
  {
    href: "/notifications",
    title: "消息中心",
    description: "查看系统通知、审批提醒、预约提醒和消息已读状态。",
    icon: Bell,
  },
  {
    href: "/settings/profile",
    title: "个人信息",
    description: "查看姓名、手机号和所属部门/实验室。",
    icon: UserRound,
  },
  {
    href: "/settings/account",
    title: "账户设置",
    description: "修改密码并管理登录会话。",
    icon: ShieldCheck,
  },
];

const trainingModules: ModuleEntry[] = [
  {
    href: "/training",
    title: "培训与准入总览",
    description: "查看课程、授权和即将到期的准入记录。",
    icon: GraduationCap,
  },
  {
    href: "/training/courses",
    title: "课程管理",
    description: "维护仪器培训、安全培训和平台使用课程。",
    icon: ShieldCheck,
  },
  {
    href: "/training/authorizations",
    title: "授权记录",
    description: "跟踪用户授权、有效期和撤销状态。",
    icon: ClipboardCheck,
  },
  {
    href: "/training/exams",
    title: "在线考试",
    description: "查看题库、提交考试记录和考试状态。",
    icon: ClipboardCheck,
  },
];

const spaceModules: ModuleEntry[] = [
  {
    href: "/spaces",
    title: "空间资源",
    description: "查看实验空间、会议室和样品前处理区，并进入预约。",
    icon: Building2,
  },
];

const extensionModules: ModuleEntry[] = [
  {
    href: "/lims/tasks",
    title: "LIMS 检测任务",
    description: "登记样本、分派任务并追踪检测结果。",
    icon: PackageCheck,
  },
  {
    href: "/eln/records",
    title: "ELN 实验记录",
    description: "维护实验记录、签名和关联任务。",
    icon: FileText,
  },
  {
    href: "/samples",
    title: "样本管理",
    description: "查看样本台账、位置和流转记录。",
    icon: TestTube2,
  },
  {
    href: "/iot/devices",
    title: "IoT 设备中心",
    description: "维护采集终端、仪器绑定和在线状态。",
    icon: Cpu,
  },
  {
    href: "/ai-assistant",
    title: "AI 助手",
    description: "基于当前租户数据回答预约、培训和运维问题。",
    icon: Bot,
  },
  {
    href: "/data-center",
    title: "数据中台",
    description: "统一查看实验室运行、资源和扩展数据汇总。",
    icon: Database,
  },
];

const workflowModules: ModuleEntry[] = [
  {
    href: "/dashboard",
    title: "我的申请",
    description: "聚合我的预约、产品申领、产品申购和待处理事项。",
    icon: LayoutDashboard,
  },
  {
    href: "/reservations",
    title: "预约记录",
    description: "查看当前账号可访问的预约记录、审批状态和使用情况。",
    icon: CalendarCheck2,
  },
  {
    href: "/approvals",
    title: "审批中心",
    description: "处理预约、产品申领和申购审批。",
    icon: ClipboardCheck,
  },
];

const financeModule: ModuleEntry = {
  href: "/finance",
  title: "财务管理",
  description: "查看使用费用、账单、费用记录、账户额度和调整流水。",
  icon: Wallet,
};

const operationAdminModules: ModuleEntry[] = [
  {
    href: "/admin",
    title: "工作概览",
    description: "查看今日预约、待审批、使用中和已履约等运营事项。",
    icon: Settings2,
    adminSection: "overview",
  },
  {
    href: "/operations",
    title: "运营看板",
    description: "查看平台运行、趋势、异常和导出报表。",
    icon: MonitorPlay,
    adminSection: "operations",
  },
  {
    href: "/admin/analytics",
    title: "运营分析中心",
    description: "观察仪器、耗材、审批、财务和风险预警分析。",
    icon: BarChart3,
    adminSection: "analytics",
  },
  {
    href: "/admin/notifications",
    title: "通知管理",
    description: "发布公告、管理系统通知和消息模板能力。",
    icon: Bell,
    adminSection: "notifications",
  },
  {
    href: "/admin/security",
    title: "安全审计与合规",
    description: "查看登录、操作、数据、权限和异常访问审计。",
    icon: ShieldCheck,
    adminSection: "security",
  },
];

const resourceAdminModules: ModuleEntry[] = [
  {
    href: "/admin/instruments",
    title: "仪器资源后台",
    description: "维护仪器档案、状态、预约规则、门禁绑定和维护记录。",
    icon: ThermometerSun,
    adminSection: "instruments",
  },
  {
    href: "/admin/materials",
    title: "资源管理后台",
    description: "维护一级目录、二级目录，并进入标准品、试剂和耗材独立管理页。",
    icon: PackageSearch,
    adminSection: "materials",
  },
  {
    href: "/maintenance",
    title: "工单与设备维护",
    description: "安排维护窗口，并联动锁定或取消受影响预约。",
    icon: Wrench,
    adminSection: "maintenance",
  },
];

const organizationAdminModules: ModuleEntry[] = [
  {
    href: "/admin/users",
    title: "用户管理",
    description: "审核账号，维护邮箱、手机号、部门、多机构归属和角色。",
    icon: UsersRound,
    adminSection: "users",
  },
];

const platformAdminModules: ModuleEntry[] = [
  {
    href: "/admin/settings",
    title: "平台配置中心",
    description: "维护系统基础配置、租户配置、通知通道和第三方集成。",
    icon: Settings2,
    adminSection: "settings",
  },
  {
    href: "/admin/settings/copy",
    title: "文案中心",
    description: "维护顶部导航、按钮、标题、占位符和首页入口文案。",
    icon: Type,
    adminSection: "settings",
  },
];

export default async function HomePage() {
  const [currentUser, copySettings] = await Promise.all([
    api.currentUserOptional(),
    api.copySettings().catch(() => createDefaultCopySettings()),
  ]);
  const t: CopyTranslator = (key, fallback = key) => copyText(copySettings, key, fallback);
  const isAdmin = isAnyAdminRole(currentUser?.role);
  const canReview = isTenantAdminRole(currentUser?.role) || currentUser?.role === "group_leader";
  const showFinance = currentUser?.role === "super_admin" || currentUser?.financeEnabled === true;
  const visibleAdminModules = (modules: ModuleEntry[]) =>
    isAdmin ? modules.filter((module) => module.adminSection && canAccessAdminSection(currentUser?.role, module.adminSection, currentUser?.financeEnabled === true)) : [];
  const publicResourceModules = resourceModules.filter((module) => module.href === "/instruments" || module.href === "/materials" || module.href === "/materials/standards" || module.href === "/materials/reagents" || module.href === "/materials/consumables");
  const resourceCatalogModules = [resourceModules[1], resourceModules[2], resourceModules[3], resourceModules[4]];
  const userWorkflowModules = canReview ? workflowModules : workflowModules.filter((module) => module.href !== "/approvals");
  const adminGroups: ModuleGroup[] = [
    {
      title: "管理员工作台",
      description: "当前为管理员前台，优先展示工作概览、运营分析、通知管理和审计入口。",
      modules: visibleAdminModules(operationAdminModules),
    },
    {
      title: "资源与准入管理",
      description: "集中进入仪器、耗材、培训、门禁绑定、预约规则和设备维护后台。",
      modules: visibleAdminModules(resourceAdminModules),
    },
    {
      title: "组织与配置",
      description: "维护用户、机构归属、权限分配、租户配置、文案、通知通道和系统基础配置。",
      modules: visibleAdminModules([...organizationAdminModules, ...platformAdminModules]),
    },
    ...(currentUser && showFinance
      ? [
          {
            title: "财务与计费中心",
            description: "进入费用流水、账单、个人账户和财务调整模块。",
            modules: [financeModule],
          },
        ]
      : []),
    {
      title: "前台常用入口",
      description: "保留仪器预约、资源目录、预约记录和消息中心，便于管理员切换到普通用户视角核对流程。",
      modules: [resourceModules[0], ...resourceCatalogModules, workflowModules[1], accessModules[0]],
    },
  ];
  const userGroups: ModuleGroup[] = [
    {
      title: "个人工作台",
      description: "当前为普通用户前台，展示个人申请、预约履约、通知、资料和账户安全入口。",
      modules: [...userWorkflowModules, ...accessModules],
    },
    {
      title: "实验资源中心",
      description: "仪器、标准品、试剂和耗材的统一前台入口，覆盖查询、预约、申领和申购。",
      modules: resourceModules,
    },
    {
      title: "培训与准入中心",
      description: "课程、授权和到期提醒入口。",
      modules: trainingModules,
    },
    {
      title: "空间资源中心",
      description: "实验空间、会议室和样品前处理间入口。",
      modules: spaceModules,
    },
    {
      title: "扩展能力中心",
      description: "LIMS、ELN、样本、IoT 和 AI 助手入口。",
      modules: extensionModules,
    },
    ...(currentUser && showFinance
      ? [
          {
            title: "财务与计费中心",
            description: "费用流水、账单、个人账户和财务调整。",
            modules: [financeModule],
          },
        ]
      : []),
  ];
  const guestGroups: ModuleGroup[] = [
    {
      title: "普通用户入口",
      description: "未登录时仅展示普通用户可进入的仪器预约和资源目录入口。",
      modules: publicResourceModules,
    },
  ];
  const visibleGroups = (currentUser ? (isAdmin ? adminGroups : userGroups) : guestGroups).filter((group) => group.modules.length > 0);
  const modeLabel = !currentUser ? "普通用户入口" : isAdmin ? "管理员工作台" : "个人工作台";
  const leadText = !currentUser
    ? "未登录时仅展示仪器预约和资源目录等普通用户入口。"
    : isAdmin
      ? "管理员前台优先展示管理中心和后台功能，同时保留前台常用入口。"
      : "普通用户前台展示个人预约、资源申领、通知、培训和账户入口。";

  return (
    <AppShell currentUser={currentUser} mainClassName="mx-auto w-full max-w-7xl px-3 pt-5 pb-4 sm:px-6 sm:pt-8 sm:pb-4 lg:px-8">
      <section className="mb-8">
        <div className="flex flex-col justify-between gap-4 md:flex-row md:items-end">
          <div className="min-w-0">
            <p className="text-xs font-bold uppercase tracking-widest text-primary">{t(modeLabel)}</p>
            <h1 className="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">{t("实验室运营系统")}</h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
              {t(leadText)}
            </p>
          </div>
        </div>
      </section>

      <div className="space-y-8">
        {visibleGroups.map((group) => (
          <ModuleGroupSection group={group} key={group.title} t={t} />
        ))}
      </div>
    </AppShell>
  );
}

function ModuleGroupSection({ group, t }: { group: ModuleGroup; t: CopyTranslator }) {
  return (
    <section>
      <div className="mb-3">
        <p className="text-xs font-bold uppercase tracking-widest text-primary">{t(group.title)}</p>
        <p className="mt-1 max-w-3xl text-sm leading-6 text-muted-foreground">{t(group.description)}</p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {group.modules.map((module) => (
          <ModuleCard entry={module} key={module.href} t={t} />
        ))}
      </div>
    </section>
  );
}

function ModuleCard({ entry, t }: { entry: ModuleEntry; t: CopyTranslator }) {
  const Icon = entry.icon;
  return (
    <Link className="group block min-w-0 rounded-lg border bg-white p-4 transition hover:border-primary/40 hover:shadow-md sm:p-5" href={entry.href}>
      <div className="flex items-start gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary transition group-hover:bg-primary group-hover:text-white">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <h2 className="break-words text-base font-bold text-slate-900">{t(entry.title)}</h2>
          <p className="mt-2 break-words text-sm leading-6 text-slate-500">{t(entry.description)}</p>
        </div>
      </div>
    </Link>
  );
}
