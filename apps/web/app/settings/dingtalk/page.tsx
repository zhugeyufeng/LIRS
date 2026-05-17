import { redirect } from "next/navigation";
import { DingTalkBindingClient } from "@/components/dingtalk-binding-client";
import { SettingsShell } from "@/components/settings-shell";
import { api } from "@/lib/api";

export default async function DingTalkBindingPage() {
  const currentUser = await api.me().catch(() => null);
  if (!currentUser) {
    redirect("/login");
  }
  const binding = await api.dingTalkBinding();

  return (
    <SettingsShell active="dingtalk" currentUser={currentUser} title="钉钉绑定" description="通过钉钉扫码授权绑定当前账号，绑定后个人通知会推送到钉钉企业应用。">
      <DingTalkBindingClient binding={binding} />
    </SettingsShell>
  );
}
