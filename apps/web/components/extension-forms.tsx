"use client";

import { FormEvent, startTransition, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { PlusCircle, Save } from "lucide-react";
import {
  browserPost,
  type AssistantQuery,
  type ElnRecord,
  type ElnRecordPayload,
  type IotDevice,
  type IotDevicePayload,
  type Instrument,
  type LimsTask,
  type LimsTaskPayload,
  type Sample,
  type SamplePayload,
  type Space,
  type SpacePayload,
  type SpaceReservation,
  type SpaceReservationPayload,
  type TrainingAuthorization,
  type TrainingAuthorizationPayload,
  type TrainingCourse,
  type TrainingCoursePayload,
  type TrainingExam,
  type TrainingExamPayload,
  type TrainingPractical,
  type TrainingPracticalPayload,
  type TrainingQuestion,
  type TrainingQuestionPayload,
  type TrainingRule,
  type TrainingRulePayload,
} from "@/lib/api";
import { Button } from "@/components/ui/button";

type ActorProps = {
  actorName: string;
};

export function TrainingCourseForm({ actorName, instruments }: ActorProps & { instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingCoursePayload = {
      title: String(form.get("title") ?? ""),
      category: String(form.get("category") ?? "仪器培训"),
      instrumentId: String(form.get("instrumentId") ?? "") || undefined,
      instructor: String(form.get("instructor") ?? actorName),
      deliveryMode: String(form.get("deliveryMode") ?? "blended"),
      durationHours: Number(form.get("durationHours") ?? 0),
      requiredForBooking: form.get("requiredForBooking") === "on",
      status: String(form.get("status") ?? "active"),
      description: String(form.get("description") ?? ""),
    };
    try {
      await browserPost<TrainingCourse>("/api/training/courses", payload);
      setMessage("课程已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field label="课程标题" name="title" placeholder="例如：高分辨率质谱仪准入培训" required />
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue="仪器培训" label="课程分类" name="category" placeholder="例如：仪器培训" />
        <Field defaultValue={actorName} label="讲师/负责人" name="instructor" placeholder="输入讲师名称" />
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <SelectField label="关联仪器" name="instrumentId">
          <option value="">无关联仪器</option>
          {instruments.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </SelectField>
        <SelectField defaultValue="blended" label="授课方式" name="deliveryMode">
          <option value="online">线上</option>
          <option value="offline">线下</option>
          <option value="blended">混合</option>
        </SelectField>
        <Field defaultValue="2" label="课程时长(小时)" min={0} name="durationHours" required step="0.5" type="number" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="active" label="状态" name="status">
          <option value="draft">草稿</option>
          <option value="active">启用</option>
          <option value="archived">归档</option>
        </SelectField>
        <label className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
          <input defaultChecked type="checkbox" name="requiredForBooking" />
          <span>预约前必须完成</span>
        </label>
      </div>
      <TextAreaField label="课程说明" name="description" placeholder="填写培训内容、准入要求和考试说明" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存课程"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function TrainingAuthorizationForm({
  actorName,
  courses,
  instruments,
}: ActorProps & { courses: TrainingCourse[]; instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingAuthorizationPayload = {
      userName: String(form.get("userName") ?? actorName),
      userId: String(form.get("userId") ?? "") || undefined,
      courseId: String(form.get("courseId") ?? "") || undefined,
      instrumentId: String(form.get("instrumentId") ?? "") || undefined,
      status: String(form.get("status") ?? "pending"),
      expiresAt: toIsoDateTime(form.get("expiresAt")),
      notes: String(form.get("notes") ?? ""),
    };
    try {
      await browserPost<TrainingAuthorization>("/api/training/authorizations", payload);
      setMessage("授权已提交");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field defaultValue={actorName} label="申请人" name="userName" placeholder="输入申请人姓名" required />
      <Field label="用户 ID（可选）" name="userId" placeholder="如果需要精确关联，可填写用户 ID" />
      <SelectField label="课程" name="courseId">
        <option value="">选择课程</option>
        {courses.map((course) => (
          <option key={course.id} value={course.id}>
            {course.title}
          </option>
        ))}
      </SelectField>
      <SelectField label="关联仪器" name="instrumentId">
        <option value="">无关联仪器</option>
        {instruments.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </SelectField>
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="pending" label="状态" name="status">
          <option value="pending">待审核</option>
          <option value="active">已授权</option>
          <option value="expired">已过期</option>
          <option value="revoked">已撤销</option>
        </SelectField>
        <Field label="到期时间" name="expiresAt" required type="datetime-local" />
      </div>
      <TextAreaField label="备注" name="notes" placeholder="填写授权说明或限制条件" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "提交中..." : "提交授权"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function TrainingQuestionForm({ actorName }: ActorProps) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingQuestionPayload = {
      title: String(form.get("title") ?? ""),
      questionType: String(form.get("questionType") ?? "single"),
      options: String(form.get("options") ?? ""),
      correctAnswer: String(form.get("correctAnswer") ?? ""),
      explanation: String(form.get("explanation") ?? ""),
      status: String(form.get("status") ?? "active"),
    };
    try {
      await browserPost<TrainingQuestion>("/api/training/questions", payload);
      setMessage("题目已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <p className="text-xs text-muted-foreground">维护人：{actorName}</p>
      <Field label="题目" name="title" placeholder="填写题目内容" required />
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="single" label="题型" name="questionType">
          <option value="single">单选</option>
          <option value="multiple">多选</option>
          <option value="judge">判断</option>
          <option value="short">简答</option>
        </SelectField>
        <SelectField defaultValue="active" label="状态" name="status">
          <option value="active">启用</option>
          <option value="draft">草稿</option>
          <option value="archived">归档</option>
        </SelectField>
      </div>
      <TextAreaField label="选项" name="options" placeholder="每行一个选项，例如：A. 需要" />
      <Field label="正确答案" name="correctAnswer" placeholder="例如：A 或 A,B" />
      <TextAreaField label="解析" name="explanation" placeholder="填写答案解析或考核要点" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存题目"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function TrainingExamForm({ actorName, courses }: ActorProps & { courses: TrainingCourse[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingExamPayload = {
      userName: String(form.get("userName") ?? actorName),
      userId: String(form.get("userId") ?? "") || undefined,
      courseId: String(form.get("courseId") ?? "") || undefined,
      score: Number(form.get("score") ?? 0),
      passed: form.get("passed") === "on",
      answers: String(form.get("answers") ?? ""),
      status: String(form.get("status") ?? "submitted"),
      notes: String(form.get("notes") ?? ""),
      examAt: toIsoDateTime(form.get("examAt")),
    };
    try {
      await browserPost<TrainingExam>("/api/training/exams", payload);
      setMessage("考试记录已提交");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field defaultValue={actorName} label="考生姓名" name="userName" placeholder="考生姓名" required />
      <Field label="用户 ID（可选）" name="userId" placeholder="管理员精确关联用户时填写" />
      <SelectField label="关联课程" name="courseId">
        <option value="">不关联课程</option>
        {courses.map((course) => (
          <option key={course.id} value={course.id}>
            {course.title}
          </option>
        ))}
      </SelectField>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="考试时间" name="examAt" required type="datetime-local" />
        <Field defaultValue="0" label="得分" min={0} name="score" required step="0.1" type="number" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="submitted" label="状态" name="status">
          <option value="draft">草稿</option>
          <option value="submitted">已提交</option>
          <option value="graded">已评分</option>
          <option value="archived">归档</option>
        </SelectField>
        <label className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
          <input type="checkbox" name="passed" />
          <span>已通过</span>
        </label>
      </div>
      <TextAreaField label="答题记录" name="answers" placeholder="填写本次答题记录或系统自动组卷结果" />
      <TextAreaField label="备注" name="notes" placeholder="填写考试说明、复核意见或异常情况" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "提交中..." : "提交考试记录"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function TrainingPracticalForm({ actorName, instruments }: ActorProps & { instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingPracticalPayload = {
      userName: String(form.get("userName") ?? ""),
      userId: String(form.get("userId") ?? "") || undefined,
      instrumentId: String(form.get("instrumentId") ?? "") || undefined,
      assessor: String(form.get("assessor") ?? actorName),
      score: Number(form.get("score") ?? 0),
      result: String(form.get("result") ?? "pending"),
      notes: String(form.get("notes") ?? ""),
      assessmentAt: toIsoDateTime(form.get("assessmentAt")),
    };
    try {
      await browserPost<TrainingPractical>("/api/training/practicals", payload);
      setMessage("实操考核已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="受考核人" name="userName" placeholder="输入用户姓名" required />
        <Field label="用户 ID（可选）" name="userId" placeholder="精确关联用户时填写" />
      </div>
      <SelectField label="关联仪器" name="instrumentId">
        <option value="">选择仪器</option>
        {instruments.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </SelectField>
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue={actorName} label="考核人" name="assessor" placeholder="输入考核人" />
        <Field label="考核时间" name="assessmentAt" required type="datetime-local" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue="0" label="得分" min={0} name="score" required step="0.1" type="number" />
        <SelectField defaultValue="pending" label="结果" name="result">
          <option value="pending">待确认</option>
          <option value="pass">通过</option>
          <option value="fail">未通过</option>
        </SelectField>
      </div>
      <TextAreaField label="考核备注" name="notes" placeholder="填写实操项目、扣分点和准入建议" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存考核"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function TrainingRuleForm({ instruments }: { instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: TrainingRulePayload = {
      instrumentId: String(form.get("instrumentId") ?? ""),
      requireTraining: form.get("requireTraining") === "on",
      requireExam: form.get("requireExam") === "on",
      requireApproval: form.get("requireApproval") === "on",
      minScore: Number(form.get("minScore") ?? 0),
      status: String(form.get("status") ?? "active"),
      notes: String(form.get("notes") ?? ""),
    };
    try {
      await browserPost<TrainingRule>("/api/training/rules", payload);
      setMessage("准入规则已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <SelectField label="关联仪器" name="instrumentId">
        <option value="">选择仪器</option>
        {instruments.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </SelectField>
      <div className="grid gap-2 text-sm">
        <label className="flex items-center gap-2 rounded-md border px-3 py-2">
          <input defaultChecked type="checkbox" name="requireTraining" />
          <span>必须完成培训</span>
        </label>
        <label className="flex items-center gap-2 rounded-md border px-3 py-2">
          <input defaultChecked type="checkbox" name="requireExam" />
          <span>必须通过考试</span>
        </label>
        <label className="flex items-center gap-2 rounded-md border px-3 py-2">
          <input defaultChecked type="checkbox" name="requireApproval" />
          <span>预约仍需审批</span>
        </label>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue="80" label="最低分" min={0} name="minScore" required step="0.1" type="number" />
        <SelectField defaultValue="active" label="状态" name="status">
          <option value="active">启用</option>
          <option value="disabled">停用</option>
        </SelectField>
      </div>
      <TextAreaField label="规则说明" name="notes" placeholder="填写培训、考试、审批和例外规则说明" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存规则"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function SpaceForm({ actorName }: ActorProps) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: SpacePayload = {
      name: String(form.get("name") ?? ""),
      kind: String(form.get("kind") ?? "lab"),
      department: String(form.get("department") ?? ""),
      location: String(form.get("location") ?? ""),
      capacity: Number(form.get("capacity") ?? 1),
      status: String(form.get("status") ?? "available"),
      accessControlPoint: String(form.get("accessControlPoint") ?? ""),
      description: String(form.get("description") ?? ""),
    };
    try {
      await browserPost<Space>("/api/spaces", payload);
      setMessage("空间已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field label="空间名称" name="name" placeholder="例如：公共会议室" required />
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="lab" label="空间类型" name="kind">
          <option value="lab">实验空间</option>
          <option value="meeting_room">会议室</option>
          <option value="workspace">工位</option>
          <option value="storage">存储间</option>
          <option value="other">其他</option>
        </SelectField>
        <Field label="归属部门" name="department" placeholder="例如：化学与分子工程学院" />
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <Field label="位置" name="location" placeholder="例如：A1-201" required />
        <Field defaultValue="1" label="容量" min={1} name="capacity" required type="number" />
        <SelectField defaultValue="available" label="状态" name="status">
          <option value="available">可用</option>
          <option value="busy">占用</option>
          <option value="maintenance">维护中</option>
          <option value="disabled">停用</option>
        </SelectField>
      </div>
      <Field label="门禁点位" name="accessControlPoint" placeholder="A1-201-DOOR" />
      <TextAreaField label="空间说明" name="description" placeholder="填写空间用途、预约规则和管理说明" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存空间"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function SpaceReservationForm({ actorName, space }: ActorProps & { space: Space }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: SpaceReservationPayload = {
      spaceId: space.id,
      requester: actorName,
      purpose: String(form.get("purpose") ?? ""),
      startTime: toIsoDateTime(form.get("startTime")),
      endTime: toIsoDateTime(form.get("endTime")),
    };
    try {
      const result = await browserPost<SpaceReservation>("/api/space-reservations", payload);
      setMessage(`预约已提交：${result.status}`);
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field defaultValue={space.name} label="预约空间" name="spaceName" disabled />
      <Field label="预约用途" name="purpose" placeholder="填写会议、培训或前处理用途" required />
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="开始时间" name="startTime" required type="datetime-local" />
        <Field label="结束时间" name="endTime" required type="datetime-local" />
      </div>
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <PlusCircle className="h-4 w-4" aria-hidden="true" />
          {pending ? "提交中..." : "提交预约"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function SampleForm({ actorName }: ActorProps) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: SamplePayload = {
      code: String(form.get("code") ?? ""),
      name: String(form.get("name") ?? ""),
      ownerName: String(form.get("ownerName") ?? actorName),
      ownerId: String(form.get("ownerId") ?? "") || undefined,
      department: String(form.get("department") ?? ""),
      groupName: String(form.get("groupName") ?? ""),
      location: String(form.get("location") ?? ""),
      status: String(form.get("status") ?? "stored"),
      hazardLevel: String(form.get("hazardLevel") ?? "normal"),
      storageCondition: String(form.get("storageCondition") ?? ""),
      description: String(form.get("description") ?? ""),
    };
    try {
      await browserPost<Sample>("/api/samples", payload);
      setMessage("样本已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="样本编号" name="code" placeholder="SMP-2026-0001" required />
        <Field label="样本名称" name="name" placeholder="填写样本名称" required />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue={actorName} label="负责人" name="ownerName" placeholder="样本负责人" />
        <Field label="负责人 ID（可选）" name="ownerId" placeholder="如果需要精确关联，可填写用户 ID" />
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <Field label="部门" name="department" placeholder="样本归属部门" />
        <Field label="课题组" name="groupName" placeholder="样本归属课题组" />
        <Field label="位置" name="location" placeholder="存储位置" />
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <SelectField defaultValue="stored" label="状态" name="status">
          <option value="stored">入库</option>
          <option value="testing">检测中</option>
          <option value="checked_out">外借</option>
          <option value="archived">归档</option>
          <option value="disposed">销毁</option>
        </SelectField>
        <SelectField defaultValue="normal" label="风险等级" name="hazardLevel">
          <option value="normal">普通</option>
          <option value="warning">警示</option>
          <option value="danger">高危</option>
        </SelectField>
        <Field label="保存条件" name="storageCondition" placeholder="例如：-80°C" />
      </div>
      <TextAreaField label="说明" name="description" placeholder="填写样本用途、流转信息和备注" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存样本"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function LimsTaskForm({ actorName, samples, instruments }: ActorProps & { samples: Sample[]; instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: LimsTaskPayload = {
      sampleId: String(form.get("sampleId") ?? "") || undefined,
      instrumentId: String(form.get("instrumentId") ?? "") || undefined,
      title: String(form.get("title") ?? ""),
      assayType: String(form.get("assayType") ?? ""),
      priority: String(form.get("priority") ?? "normal"),
      status: String(form.get("status") ?? "pending"),
      requesterName: String(form.get("requesterName") ?? actorName),
      dueAt: toIsoDateTime(form.get("dueAt")),
      resultSummary: String(form.get("resultSummary") ?? ""),
    };
    try {
      await browserPost<LimsTask>("/api/lims/tasks", payload);
      setMessage("LIMS 任务已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field label="任务标题" name="title" placeholder="填写检测任务标题" required />
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField label="关联样本" name="sampleId">
          <option value="">无关联样本</option>
          {samples.map((sample) => (
            <option key={sample.id} value={sample.id}>
              {sample.code} / {sample.name}
            </option>
          ))}
        </SelectField>
        <SelectField label="关联仪器" name="instrumentId">
          <option value="">无关联仪器</option>
          {instruments.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </SelectField>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <Field label="检测类型" name="assayType" placeholder="例如：流式检测" />
        <SelectField defaultValue="normal" label="优先级" name="priority">
          <option value="normal">普通</option>
          <option value="high">高</option>
          <option value="urgent">紧急</option>
        </SelectField>
        <Field defaultValue={actorName} label="申请人" name="requesterName" placeholder="输入申请人" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="截止时间" name="dueAt" required type="datetime-local" />
        <SelectField defaultValue="pending" label="状态" name="status">
          <option value="pending">待分配</option>
          <option value="assigned">已分配</option>
          <option value="running">进行中</option>
          <option value="completed">已完成</option>
          <option value="cancelled">已取消</option>
        </SelectField>
      </div>
      <TextAreaField label="结果摘要" name="resultSummary" placeholder="填写预计结果或当前说明" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存任务"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function ElnRecordForm({ actorName, tasks }: ActorProps & { tasks: LimsTask[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: ElnRecordPayload = {
      title: String(form.get("title") ?? ""),
      authorName: String(form.get("authorName") ?? actorName),
      authorId: String(form.get("authorId") ?? "") || undefined,
      project: String(form.get("project") ?? ""),
      linkedTaskId: String(form.get("linkedTaskId") ?? "") || undefined,
      content: String(form.get("content") ?? ""),
      status: String(form.get("status") ?? "draft"),
    };
    try {
      await browserPost<ElnRecord>("/api/eln/records", payload);
      setMessage("ELN 记录已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field label="记录标题" name="title" placeholder="填写实验记录标题" required />
      <div className="grid gap-3 md:grid-cols-2">
        <Field defaultValue={actorName} label="作者" name="authorName" placeholder="输入作者" />
        <Field label="作者 ID（可选）" name="authorId" placeholder="如果需要精确关联，可填写用户 ID" />
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="项目/课题" name="project" placeholder="填写项目名称" />
        <SelectField label="关联 LIMS 任务" name="linkedTaskId">
          <option value="">无关联任务</option>
          {tasks.map((task) => (
            <option key={task.id} value={task.id}>
              {task.title}
            </option>
          ))}
        </SelectField>
      </div>
      <SelectField defaultValue="draft" label="状态" name="status">
        <option value="draft">草稿</option>
        <option value="submitted">已提交</option>
        <option value="signed">已签名</option>
        <option value="archived">已归档</option>
      </SelectField>
      <TextAreaField label="实验内容" name="content" placeholder="填写实验步骤、附件位置和原始数据说明" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存记录"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function IotDeviceForm({ actorName, instruments }: ActorProps & { instruments: Instrument[] }) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload: IotDevicePayload = {
      name: String(form.get("name") ?? ""),
      vendor: String(form.get("vendor") ?? ""),
      deviceCode: String(form.get("deviceCode") ?? ""),
      instrumentId: String(form.get("instrumentId") ?? "") || undefined,
      online: form.get("online") === "on",
      status: String(form.get("status") ?? "offline"),
      telemetry: String(form.get("telemetry") ?? "{}"),
      notes: String(form.get("notes") ?? ""),
    };
    try {
      await browserPost<IotDevice>("/api/iot/devices", payload);
      setMessage("IoT 设备已保存");
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "保存失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Field label="设备名称" name="name" placeholder="例如：质谱采集终端" required />
      <div className="grid gap-3 md:grid-cols-2">
        <Field label="厂商" name="vendor" placeholder="例如：LIRS-IoT" />
        <Field label="设备编码" name="deviceCode" placeholder="例如：IOT-0001" />
      </div>
      <SelectField label="关联仪器" name="instrumentId">
        <option value="">无关联仪器</option>
        {instruments.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </SelectField>
      <div className="grid gap-3 md:grid-cols-2">
        <SelectField defaultValue="offline" label="状态" name="status">
          <option value="online">在线</option>
          <option value="offline">离线</option>
          <option value="warning">预警</option>
          <option value="disabled">停用</option>
        </SelectField>
        <label className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
          <input defaultChecked type="checkbox" name="online" />
          <span>当前在线</span>
        </label>
      </div>
      <TextAreaField defaultValue="{}" label="遥测数据(JSON)" name="telemetry" placeholder='例如：{"temperature":"22.4"}' />
      <TextAreaField label="备注" name="notes" placeholder="填写接入说明或网关备注" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "保存中..." : "保存设备"}
        </Button>
      </div>
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

export function AssistantQueryForm({ actorName }: ActorProps) {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [answer, setAnswer] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const payload = {
      question: String(form.get("question") ?? ""),
      context: String(form.get("context") ?? ""),
    };
    try {
      const result = await browserPost<AssistantQuery>("/api/ai-assistant", payload);
      setMessage("已生成回答");
      setAnswer(result.answer);
      formElement.reset();
      startTransition(() => router.refresh());
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "发送失败");
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <p className="text-xs text-muted-foreground">当前账号：{actorName}</p>
      <TextAreaField label="问题描述" name="question" placeholder="例如：今天有哪些预约需要处理？" required />
      <Field label="问题背景（可选）" name="context" placeholder="可填写仪器、样本或流程背景" />
      <div className="flex justify-end">
        <Button disabled={pending} type="submit">
          <Save className="h-4 w-4" aria-hidden="true" />
          {pending ? "生成中..." : "发送问题"}
        </Button>
      </div>
      {answer ? <div className="rounded-md border bg-slate-50 p-3 text-sm leading-6 text-slate-700">{answer}</div> : null}
      {message ? <p className="text-xs text-muted-foreground">{message}</p> : null}
    </form>
  );
}

function Field({
  defaultValue,
  label,
  name,
  placeholder,
  required,
  type = "text",
  disabled,
  min,
  step,
}: {
  defaultValue?: string;
  label: string;
  name: string;
  placeholder?: string;
  required?: boolean;
  type?: string;
  disabled?: boolean;
  min?: number;
  step?: string | number;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <input
        className="h-10 w-full rounded-md border bg-white px-3 text-sm"
        defaultValue={defaultValue}
        disabled={disabled}
        min={min}
        name={name}
        placeholder={placeholder}
        required={required}
        step={step}
        type={type}
      />
    </label>
  );
}

function SelectField({
  children,
  defaultValue,
  label,
  name,
}: {
  children: ReactNode;
  defaultValue?: string;
  label: string;
  name: string;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <select className="h-10 w-full rounded-md border bg-white px-3 text-sm" defaultValue={defaultValue} name={name}>
        {children}
      </select>
    </label>
  );
}

function TextAreaField({
  defaultValue,
  label,
  name,
  placeholder,
  required,
}: {
  defaultValue?: string;
  label: string;
  name: string;
  placeholder?: string;
  required?: boolean;
}) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <textarea className="min-h-24 w-full rounded-md border bg-white px-3 py-2 text-sm" defaultValue={defaultValue} name={name} placeholder={placeholder} required={required} />
    </label>
  );
}

function toIsoDateTime(value: FormDataEntryValue | null) {
  const date = new Date(String(value ?? ""));
  return Number.isNaN(date.getTime()) ? "" : date.toISOString();
}
