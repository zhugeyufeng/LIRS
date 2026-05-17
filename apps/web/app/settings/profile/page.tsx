import { redirect } from "next/navigation";
import { ProfileSettingsForm } from "@/components/account-settings-forms";
import { SettingsShell } from "@/components/settings-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";

export default async function ProfileSettingsPage() {
  const currentUser = await api.me().catch(() => null);
  if (!currentUser) {
    redirect("/login");
  }

  return (
    <SettingsShell active="profile" currentUser={currentUser} title="个人资料" description="查看当前账号身份信息，姓名、手机号和所属部门由管理员维护。">
      <Card>
        <CardHeader>
          <CardTitle>个人资料</CardTitle>
        </CardHeader>
        <CardContent>
          <ProfileSettingsForm user={currentUser} />
        </CardContent>
      </Card>
    </SettingsShell>
  );
}
