import { BellDot, Bot, Building2, Landmark, MessageSquareCode, PanelBottom, ShieldCheck, Type, Wallet } from "lucide-react";
import { AdminSettingsCard, AdminSettingsNav } from "@/components/admin-settings-nav";
import { AdminShell, requireAdminSection } from "@/components/admin-shell";

export default async function AdminSettingsPage() {
  const currentUser = await requireAdminSection("settings");

  return (
    <AdminShell active="settings" title="平台配置中心" description="按照系统基础配置、单位/机构信息、通知通道和第三方集成拆分后台设置层级。">
      <AdminSettingsNav active="overview" role={currentUser.role} />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <AdminSettingsCard href="/admin/settings/organization" icon={Building2} title="组织架构管理" description="维护单位/机构下部门、实验室和团队等基础数据，按机构独立配置。" />
        <AdminSettingsCard href="/admin/settings/tenants" icon={Landmark} title="单位/机构信息" description="维护单位名称、机构编码、机构状态和财务模块开关。" />
        <AdminSettingsCard href="/admin/settings/billing" icon={Wallet} title="财务模块开关" description="按机构启用或停用财务模块，控制财务菜单与接口访问。" />
        {currentUser.role === "super_admin" ? <AdminSettingsCard href="/admin/settings/notifications" icon={BellDot} title="通知通道配置" description="维护 Microsoft Graph 邮件，并预留微信公众号、服务号接口配置。" /> : null}
        <AdminSettingsCard href="/admin/settings/dingtalk" icon={MessageSquareCode} title="钉钉应用设置" description="按机构维护钉钉企业应用新版凭证、机器人编码、扫码绑定回调和 HTTP 事件订阅。" />
        <AdminSettingsCard href="/admin/settings/ai-assistant" icon={Bot} title="AI 模型设置" description="按机构维护 AI 助手模型、API 地址、API Key 和系统提示词。" />
        {currentUser.role === "super_admin" ? <AdminSettingsCard href="/admin/settings/access-control" icon={ShieldCheck} title="第三方集成" description="维护海康威视、大华门禁对接参数；具体仪器门禁匹配在仪器管理中设置。" /> : null}
        <AdminSettingsCard href="/admin/settings/copy" icon={Type} title="文案中心" description="维护顶部导航、按钮、标题、占位符和首页入口文案。" />
        <AdminSettingsCard href="/admin/settings/footer" icon={PanelBottom} title="系统基础配置" description="维护网站域名、全站 Logo、Footer、简介、栏目和版权信息。" />
      </div>
    </AdminShell>
  );
}
