import { PasswordSettingsForm } from "@/components/account-settings-forms";
import { SettingsShell } from "@/components/settings-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function AccountSettingsPage() {
  return (
    <SettingsShell active="account" title="账户安全" description="修改密码后会清理登录状态，需要重新登录。">
      <Card>
        <CardHeader>
          <CardTitle>修改密码</CardTitle>
        </CardHeader>
        <CardContent>
          <PasswordSettingsForm />
        </CardContent>
      </Card>
    </SettingsShell>
  );
}
