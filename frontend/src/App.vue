<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import AudioClip from "./components/AudioClip.vue";
import AudioImport from "./components/AudioImport.vue";
import {
  BookOpen,
  GraduationCap,
  Trophy,
  Flame,
  Star,
  ShieldCheck,
  ArrowRight,
  Check,
  PenLine,
  Sparkles,
  LogOut,
  UserRound,
  Home,
  ListChecks,
  Library,
  School,
  Menu,
  X,
  Download,
  Users,
  Medal,
  Crown,
  Send,
  CircleCheck,
  BookHeart,
  Heart,
  Plus,
  LockKeyhole,
  RefreshCw,
} from "lucide-vue-next";
import {
  API,
  ApiError,
  api,
  uuid,
  type User,
  type Task,
  type Progress,
  type Achievement,
  type LeaderboardEntry,
  type Case,
  type School as SchoolType,
  type Material,
  type Translation,
  type DailyQuest,
  type DailyQuestsResponse,
  type AnswerResult,
} from "./api";
type Tab =
  | "home"
  | "learn"
  | "leaderboard"
  | "library"
  | "schools"
  | "profile"
  | "moderation";
const tab = ref<Tab>("home");
const token = ref(localStorage.getItem("bashkir_token") || "");
const user = ref<User | null>(null);
const progress = ref<Progress | null>(null);
const task = ref<Task | null>(null);
const dailyQuests = ref<DailyQuest[]>([]);
const lastAnswer = ref<AnswerResult | null>(null);
const achievements = ref<Achievement[]>([]);
const leaders = ref<LeaderboardEntry[]>([]);
const schools = ref<SchoolType[]>([]);
const languagePairs = ref<LanguagePair[]>([]);
const allLanguagePairs = ref<LanguagePair[]>([]);
const chosenPair = ref("ba-ru");
const registerPair = ref("ba-ru");
const seedPair = ref("ba-ru");
const newPair = ref({
  sourceLang: "ba",
  targetLang: "en",
  name: "Башкирский → Английский",
});
type LanguagePair = {
  code: string;
  sourceLang: string;
  targetLang: string;
  name: string;
  active: boolean;
};

const materials = ref<Material[]>([]);
const translations = ref<Translation[]>([]);
const cases = ref<Case[]>([]);
const classes = ref<{ id: string; name: string; school: string }[]>([]);
const classStats = ref<Record<string, unknown> | null>(null);
const classStudents = ref<
  {
    userId: string;
    displayName: string;
    totalXp: number;
    completedTasks: number;
  }[]
>([]);
const authMode = ref<"login" | "register">("register");
const form = ref({ email: "", password: "", displayName: "" });
const correction = ref("");
const correcting = ref(false);
const completedNow = ref(false);
const toast = ref("");
const loading = ref(false);
const authError = ref("");
const mobileNav = ref(false);
const boardScope = ref("global");
const boardPeriod = ref("weekly");
const boardMetric = ref("xp");
const studentRegistration = ref({ email: "", password: "", displayName: "" });
const schoolName = ref("");
const aliasCandidate = ref("");
const aliasCanonical = ref("");
const newClassName = ref("");
const selectedSchool = ref("");
const profileSchool = ref("");
const studentToAdd = ref("");
const newMaterial = ref({
  title: "",
  type: "text",
  license: "demo / educational",
  description: "",
});
const newFragment = ref({
  materialId: "",
  textBa: "",
  sourceLang: "ba",
  chapter: "",
});
const createdFragmentId = ref("");
const createdFragmentSource = ref("ba");
const seedTranslation = ref("");
const newRoleUserId = ref("");
const newRole = ref("teacher");
const resolveCandidate = ref("");
const selectedCase = ref("");
const candidates = ref<{ id: string; translation: string; votes: number }[]>(
  [],
);
const linkCode = ref("");
type LiteratureDraft = {
  id: string;
  text: string;
  title: string;
  chapter: string;
  candidates: number;
  status: string;
  sourceLang: string;
  correct?: string | null;
  incorrect?: string | null;
  errorExplanation?: string | null;
};
const literatureDrafts = ref<LiteratureDraft[]>([]);
const literatureFile = ref<File | null>(null);
const literatureText = ref("");
const literatureSegmentMode = ref<"sentence" | "paragraph">("sentence");
const literatureRightsConfirmed = ref(false);
const literatureMeta = ref({
  title: "",
  author: "",
  license: "public-domain",
  copyrightNote: "",
  sourceLang: "ba",
});
const literaturePair = ref("ba-ru");
const literatureBusy = ref("");
const literatureAIResults = ref<
  Record<
    string,
    { correct: string; incorrect: string; errorExplanation: string }
  >
>({});
function onLiteratureFile(event: Event) {
  literatureFile.value = (event.target as HTMLInputElement).files?.[0] || null;
}
async function loadLiteratureDrafts() {
  if (user.value?.role === "admin") {
    literatureDrafts.value = await api<LiteratureDraft[]>(
      `/admin/literature/drafts?langPair=${encodeURIComponent(literaturePair.value)}`,
    );
    for (const draft of literatureDrafts.value) {
      if (!literatureAIResults.value[draft.id])
        literatureAIResults.value[draft.id] = {
          correct: draft.correct || "",
          incorrect: draft.incorrect || "",
          errorExplanation: draft.errorExplanation || "",
        };
    }
  }
}
async function uploadLiterature() {
  await safe(async () => {
    const payload = new FormData();
    for (const [key, value] of Object.entries(literatureMeta.value))
      payload.append(key, value);
    payload.append("segmentMode", literatureSegmentMode.value);
    payload.append("rightsConfirmed", String(literatureRightsConfirmed.value));
    if (literatureFile.value) payload.append("file", literatureFile.value);
    else payload.append("text", literatureText.value);
    const result = await fetch(`${API}/admin/literature/import`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token.value}` },
      body: payload,
    });
    const data = await result.json().catch(() => ({}));
    if (!result.ok)
      throw new Error(
        data?.error?.message || `Ошибка импорта ${result.status}`,
      );
    literatureMeta.value = {
      title: "",
      author: "",
      license: "public-domain",
      copyrightNote: "",
      sourceLang: "ba",
    };
    literatureText.value = "";
    literatureFile.value = null;
    literatureRightsConfirmed.value = false;
    await loadLiteratureDrafts();
    notice(`Произведение загружено: ${data.fragmentsCreated} черновиков`);
  });
}
async function generateLiterature(id: string) {
  literatureBusy.value = id;
  try {
    const result = await api<{
      correct: string;
      incorrect: string;
      errorExplanation: string;
    }>(
      `/admin/literature/fragments/${id}/generate?langPair=${encodeURIComponent(literaturePair.value)}`,
      "POST",
      {},
    );
    literatureAIResults.value[id] = result;
    await loadLiteratureDrafts();
    notice("ИИ предложил переводы. Проверьте их до публикации.");
  } catch (e) {
    notice(e instanceof Error ? e.message : "Ошибка ИИ");
  } finally {
    literatureBusy.value = "";
  }
}
async function saveLiteratureTranslations(id: string) {
  await safe(async () => {
    const translations = literatureAIResults.value[id];
    if (!translations) return;
    await api(`/admin/literature/fragments/${id}/translations`, "PUT", {
      languagePair: literaturePair.value,
      ...translations,
    });
    await loadLiteratureDrafts();
    notice("Проверенные варианты сохранены");
  });
}
async function registerClassStudent() {
  if (!selectedCase.value) return;
  await safe(async () => {
    await api(`/classes/${selectedCase.value}/register-student`, "POST", {
      ...studentRegistration.value,
      languagePair: chosenPair.value,
    });
    studentRegistration.value = { email: "", password: "", displayName: "" };
    await showClass(selectedCase.value);
    notice("Ученик зарегистрирован");
  });
}
async function createSchool() {
  await safe(async () => {
    await api("/schools", "POST", { name: schoolName.value });
    schools.value = await api("/schools");
    schoolName.value = "";
    notice("Школа создана");
  });
}
async function mergeSynonyms() {
  const item = cases.value.find((x) => x.id === selectedCase.value);
  if (!item) return;
  await safe(async () => {
    await api(`/fragments/${item.fragmentId}/equivalences`, "POST", {
      candidateId: aliasCandidate.value,
      canonicalId: aliasCanonical.value,
    });
    await loadCandidates(item.id);
    cases.value = await api("/moderation/cases");
    notice("Синонимы объединены, голоса пересчитаны");
  });
}
async function moderateLiterature(id: string, action: "publish" | "reject") {
  literatureBusy.value = id;
  try {
    await api(
      `/admin/literature/fragments/${id}/${action}`,
      "POST",
      action === "publish" ? { languagePair: literaturePair.value } : {},
    );
    await loadLiteratureDrafts();
    notice(action === "publish" ? "Фрагмент опубликован" : "Фрагмент отклонён");
  } catch (e) {
    notice(e instanceof Error ? e.message : "Ошибка модерации");
  } finally {
    literatureBusy.value = "";
  }
}
const isStaff = computed(() =>
  ["admin", "moderator"].includes(user.value?.role || ""),
);
const isTeacher = computed(() =>
  ["admin", "teacher"].includes(user.value?.role || ""),
);
const nav = computed(
  () =>
    [
      { id: "home", title: "Главная", icon: Home },
      { id: "learn", title: "Задания", icon: BookOpen },
      { id: "leaderboard", title: "Рейтинг", icon: Trophy },
      { id: "library", title: "Переводы", icon: Library },
      { id: "schools", title: "Школы", icon: School },
      { id: "profile", title: "Профиль", icon: UserRound },
      ...(isStaff.value
        ? [{ id: "moderation", title: "Модерация", icon: ShieldCheck }]
        : []),
    ] as { id: Tab; title: string; icon: typeof Home }[],
);
function notice(text: string) {
  toast.value = text;
  setTimeout(() => {
    if (toast.value === text) toast.value = "";
  }, 4400);
}
function logout() {
  localStorage.removeItem("bashkir_token");
  token.value = "";
  user.value = null;
  task.value = null;
  dailyQuests.value = [];
  tab.value = "home";
  notice("Вы вышли из аккаунта");
}
async function safe(fn: () => Promise<void>) {
  loading.value = true;
  try {
    await fn();
  } catch (e) {
    notice(e instanceof Error ? e.message : "Произошла ошибка");
  } finally {
    loading.value = false;
  }
}
async function refreshPairs() {
  languagePairs.value = await api<LanguagePair[]>("/language-pairs");
  if (isStaff.value && user.value?.role === "admin")
    allLanguagePairs.value = await api<LanguagePair[]>("/admin/language-pairs");
  if (!languagePairs.value.find((x) => x.code === registerPair.value))
    registerPair.value = languagePairs.value[0]?.code || "ba-ru";
}
async function loadProfile() {
  user.value = await api<User>("/users/me");
  chosenPair.value = user.value.languagePair;
  progress.value = await api<Progress>("/gamification/me");
}
async function createLanguagePair() {
  await safe(async () => {
    const created = await api<LanguagePair>(
      "/admin/language-pairs",
      "POST",
      newPair.value,
    );
    await refreshPairs();
    seedPair.value = created.code;
    notice("Языковая пара добавлена. Теперь загрузите для неё переводы.");
  });
}
async function togglePair(item: LanguagePair) {
  await safe(async () => {
    await api(`/admin/language-pairs/${item.code}`, "PATCH", {
      active: !item.active,
    });
    await refreshPairs();
    notice("Статус языковой пары изменён");
  });
}
async function updatePairName(item: LanguagePair) {
  const name = prompt("Новое название пары", item.name);
  if (!name || name === item.name) return;
  await safe(async () => {
    await api(`/admin/language-pairs/${item.code}`, "PATCH", { name });
    await refreshPairs();
    notice("Название изменено");
  });
}
async function savePair() {
  await safe(async () => {
    await api("/users/me", "PATCH", { languagePair: chosenPair.value });
    await loadProfile();
    task.value = null;
    if (tab.value === "learn") await loadTask();
    notice("Языковая пара сохранена");
  });
}

async function loadDailyQuests() {
  dailyQuests.value = (await api<DailyQuestsResponse>("/daily-quests")).quests;
}
async function loadAll() {
  await loadProfile();
  await loadDailyQuests();
  await refreshPairs();
  const [a, s, m, t] = await Promise.all([
    api<Achievement[]>("/gamification/achievements"),
    api<SchoolType[]>("/schools"),
    api<Material[]>("/materials"),
    api<Translation[]>("/translations"),
  ]);
  achievements.value = a;
  schools.value = s;
  profileSchool.value = user.value?.schoolId || "";
  materials.value = m;
  translations.value = t;
  if (isTeacher.value) classes.value = await api("/classes");
  if (isStaff.value) cases.value = await api("/moderation/cases");
  if (user.value?.role === "admin") await loadLiteratureDrafts();
}
async function authenticate() {
  authError.value = "";
  loading.value = true;
  try {
    const path =
      authMode.value === "register" ? "/auth/register" : "/auth/login";
    const body =
      authMode.value === "register"
        ? { ...form.value, languagePair: registerPair.value }
        : { email: form.value.email, password: form.value.password };
    const result = await api<{ accessToken: string; user: User }>(
      path,
      "POST",
      body,
    );
    localStorage.setItem("bashkir_token", result.accessToken);
    token.value = result.accessToken;
    await loadAll();
    notice("Рәхим итегеҙ! Добро пожаловать!");
  } catch (e) {
    localStorage.removeItem("bashkir_token");
    token.value = "";
    user.value = null;
    authError.value =
      e instanceof Error ? e.message : "Не удалось войти. Попробуйте ещё раз.";
  } finally {
    loading.value = false;
  }
}
async function navigate(t: Tab) {
  tab.value = t;
  mobileNav.value = false;
  if (t === "learn")
    await safe(async () => {
      await loadDailyQuests();
      await loadTask();
    });
  if (t === "home") await safe(loadDailyQuests);
  if (t === "leaderboard") await safe(loadLeaders);
  if (t === "library")
    await safe(async () => {
      translations.value = await api("/translations");
    });
  if (t === "moderation")
    await safe(async () => {
      cases.value = await api("/moderation/cases");
      await loadLiteratureDrafts();
    });
}
async function loadTask() {
  lastAnswer.value = null;
  task.value = await api<Task>(
    `/tasks/daily?langPair=${encodeURIComponent(chosenPair.value)}`,
  );
  completedNow.value = task.value.status === "completed";
  correction.value = "";
  correcting.value = false;
}
async function submitAnswer(kind: "confirmation" | "correction") {
  if (!task.value) return;
  await safe(async () => {
    const translation =
      kind === "confirmation"
        ? task.value!.suggestedTranslation
        : correction.value.trim();
    if (!translation) {
      notice("Введите перевод");
      return;
    }
    lastAnswer.value = await api<AnswerResult>(
      `/tasks/${task.value!.taskId}/answers`,
      "POST",
      { answerType: kind, translation, idempotencyKey: uuid() },
    );
    completedNow.value = true;
    await Promise.all([loadProfile(), loadDailyQuests()]);
    achievements.value = await api("/gamification/achievements");
    notice(
      lastAnswer.value.questBonusXp > 0
        ? `Цель выполнена! +${lastAnswer.value.xpEarned} XP`
        : "Перевод сохранён! +10 XP",
    );
  });
}
async function loadLeaders() {
  const res = await api<{ entries: LeaderboardEntry[] }>(
    `/leaderboards?scope=${boardScope.value}&period=${boardPeriod.value}&metric=${boardMetric.value}`,
  );
  leaders.value = res.entries;
}
async function exportData(format: "csv" | "json") {
  await safe(async () => {
    const url = `${API}/translations/export?format=${format}`;
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${token.value}` },
    });
    if (!res.ok) throw new Error("Не удалось выгрузить переводы");
    const blob = await res.blob();
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `bashkir-translations.${format}`;
    a.click();
    URL.revokeObjectURL(a.href);
  });
}
async function createClass() {
  if (!selectedSchool.value || !newClassName.value) return;
  await safe(async () => {
    await api("/classes", "POST", {
      schoolId: selectedSchool.value,
      name: newClassName.value,
    });
    classes.value = await api("/classes");
    newClassName.value = "";
    notice("Класс создан");
  });
}
async function showClass(id: string) {
  await safe(async () => {
    classStats.value = await api(`/classes/${id}/statistics`);
    classStudents.value = await api(`/classes/${id}/students`);
    selectedCase.value = id;
  });
}
async function addStudent() {
  if (!selectedCase.value || !studentToAdd.value) return;
  await safe(async () => {
    await api(`/classes/${selectedCase.value}/students`, "POST", {
      studentId: studentToAdd.value,
    });
    await showClass(selectedCase.value);
    studentToAdd.value = "";
    notice("Ученик добавлен");
  });
}
async function createMaterial() {
  await safe(async () => {
    await api("/materials", "POST", newMaterial.value);
    materials.value = await api("/materials");
    newMaterial.value = {
      title: "",
      type: "text",
      license: "demo / educational",
      description: "",
    };
    notice("Материал добавлен");
  });
}
async function createFragment() {
  await safe(async () => {
    const data = await api<{ id: string }>(
      "/fragments",
      "POST",
      newFragment.value,
    );
    createdFragmentId.value = data.id;
    createdFragmentSource.value = newFragment.value.sourceLang;
    seedPair.value =
      languagePairs.value.find(
        (p) => p.sourceLang === createdFragmentSource.value,
      )?.code || "";
    newFragment.value = {
      materialId: "",
      textBa: "",
      sourceLang: "ba",
      chapter: "",
    };
    notice("Фрагмент создан. Добавьте исходный перевод ниже.");
  });
}
async function createSeedTranslation() {
  if (!createdFragmentId.value || !seedTranslation.value.trim()) return;
  await safe(async () => {
    await api(`/fragments/${createdFragmentId.value}/candidates`, "POST", {
      translation: seedTranslation.value.trim(),
      languagePair: seedPair.value,
    });
    seedTranslation.value = "";
    await loadLiteratureDrafts();
    notice(
      "Вариант сохранён. В черновиках укажите правильный и ошибочный переводы и опубликуйте фрагмент.",
    );
  });
}
async function changeRole() {
  if (!newRoleUserId.value) return;
  await safe(async () => {
    await api(`/users/${newRoleUserId.value}/role`, "POST", {
      role: newRole.value,
    });
    newRoleUserId.value = "";
    notice("Роль обновлена");
  });
}
async function loadCandidates(id: string) {
  selectedCase.value = id;
  await safe(async () => {
    const c = cases.value.find((x) => x.id === id);
    if (!c) return;
    candidates.value = await api(
      `/fragments/${c.fragmentId}/candidates?langPair=${encodeURIComponent(c.langPair)}`,
    );
    resolveCandidate.value = candidates.value[0]?.id || "";
  });
}
async function resolve() {
  if (!resolveCandidate.value) return;
  await safe(async () => {
    await api(`/moderation/cases/${selectedCase.value}/resolve`, "POST", {
      candidateId: resolveCandidate.value,
    });
    cases.value = await api("/moderation/cases");
    translations.value = await api("/translations");
    candidates.value = [];
    selectedCase.value = "";
    notice("Перевод утверждён");
  });
}
async function updateSchool() {
  await safe(async () => {
    await api("/users/me", "PATCH", {
      schoolId: profileSchool.value || undefined,
      clearSchool: !profileSchool.value,
    });
    await loadProfile();
    notice("Школа сохранена");
  });
}
async function getLink() {
  await safe(async () => {
    const data = await api<{ code: string }>("/users/me/link-code");
    linkCode.value = data.code;
  });
}
onMounted(async () => {
  try {
    if (token.value) {
      await loadAll();
    } else {
      await refreshPairs();
    }
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) logout();
    else
      notice(
        error instanceof Error ? error.message : "Не удалось загрузить данные",
      );
  }
});
</script>
<template>
  <div v-if="!token" class="auth-layout">
    <div class="auth-hero">
      <div class="auth-hero-inner">
        <div class="brand auth-brand">
          <div class="brand-icon"><BookHeart :size="27" /></div>
          <span>телем<span class="brand-accent">күпер</span></span>
        </div>
        <div class="hero-art">
          <div class="hero-orbit orbit-one"></div>
          <div class="hero-orbit orbit-two"></div>
          <div class="hero-mascot">Ә</div>
          <div class="floating-card top">
            <Flame :size="22" /> 7 дней подряд
          </div>
          <div class="floating-card bottom"><Star :size="20" /> +10 XP</div>
        </div>
        <div class="eyebrow light">ЯЗЫК, КОТОРЫЙ НАС ОБЪЕДИНЯЕТ</div>
        <h1>Каждое слово<br />имеет <em>значение.</em></h1>
        <p>
          Изучайте башкирский, сохраняйте переводы и открывайте язык заново.
          Маленький шаг каждый день — большой вклад в будущее.
        </p>
        <div class="hero-stats">
          <div><strong>5 мин</strong><span>в день</span></div>
          <div><strong>+10 XP</strong><span>за задание</span></div>
          <div><strong>∞</strong><span>новых слов</span></div>
        </div>
      </div>
    </div>
    <div class="auth-side">
      <div class="auth-container">
        <div class="mobile-auth-logo brand">
          <div class="brand-icon"><BookHeart :size="23" /></div>
          телем<span class="brand-accent">күпер</span>
        </div>
        <div class="auth-header">
          <span class="pill pill-green"
            ><Sparkles :size="14" /> Начните бесплатно</span
          >
          <h2>
            {{
              authMode === "register"
                ? "Создайте свой аккаунт"
                : "С возвращением!"
            }}
          </h2>
          <p>
            {{
              authMode === "register"
                ? "Ваша история изучения языка начинается здесь."
                : "Продолжайте там, где остановились."
            }}
          </p>
        </div>
        <form class="auth-form" @submit.prevent="authenticate">
          <p
            v-if="authError"
            class="form-error"
            role="alert"
            aria-live="polite"
          >
            {{ authError }}
          </p>
          <label v-if="authMode === 'register'"
            >Как вас зовут?<input
              v-model="form.displayName"
              placeholder="Ваше имя"
              required
              minlength="2" /></label
          ><label
            >Электронная почта<input
              v-model="form.email"
              placeholder="student@example.com"
              type="email"
              autocomplete="email"
              required /></label
          ><label
            >Пароль<input
              v-model="form.password"
              :autocomplete="
                authMode === 'register' ? 'new-password' : 'current-password'
              "
              placeholder="Не менее 8 символов"
              type="password"
              required
              minlength="8" /></label
          ><label v-if="authMode === 'register'"
            >Направление перевода<select v-model="registerPair">
              <option
                v-for="pair in languagePairs"
                :key="pair.code"
                :value="pair.code"
              >
                {{ pair.name }}
              </option>
            </select></label
          ><button
            class="btn btn-primary large"
            :disabled="loading"
            type="submit"
          >
            {{
              loading
                ? "Подождите..."
                : authMode === "register"
                  ? "Создать аккаунт"
                  : "Войти"
            }}
            <ArrowRight :size="19" />
          </button>
        </form>
        <p class="auth-switch">
          {{
            authMode === "register" ? "Уже есть аккаунт?" : "Ещё нет аккаунта?"
          }}
          <button
            @click="
              authMode = authMode === 'register' ? 'login' : 'register';
              authError = '';
            "
          >
            {{ authMode === "register" ? "Войти" : "Зарегистрироваться" }}
          </button>
        </p>
        <div class="auth-foot">Учимся и сохраняем языки вместе</div>
      </div>
    </div>
  </div>
  <div v-else class="app-shell">
    <aside class="sidebar" :class="{ 'sidebar-open': mobileNav }">
      <div class="brand sidebar-brand" @click="navigate('home')">
        <div class="brand-icon"><BookHeart :size="26" /></div>
        <span>телем<span class="brand-accent">күпер</span></span>
      </div>
      <div class="sidebar-heading">МЕНЮ</div>
      <nav class="navigation">
        <button
          v-for="item in nav"
          :key="item.id"
          :class="['nav-item', { active: tab === item.id }]"
          @click="navigate(item.id)"
        >
          <component :is="item.icon" :size="20" /><span>{{ item.title }}</span
          ><span
            v-if="
              item.id === 'moderation' &&
              cases.filter((x) => x.status === 'open').length
            "
            class="nav-count"
            >{{ cases.filter((x) => x.status === "open").length }}</span
          >
        </button>
      </nav>
      <div class="sidebar-bottom">
        <div class="sidebar-challenge">
          <div class="challenge-top"><Flame :size="19" /> ВАША СЕРИЯ</div>
          <div class="challenge-count">
            {{ progress?.streak || 0 }} <span>дней</span>
          </div>
          <div class="challenge-caption">Возвращайтесь каждый день!</div>
        </div>
        <button class="nav-item logout" @click="logout">
          <LogOut :size="19" /><span>Выйти</span>
        </button>
      </div>
    </aside>
    <div
      class="mobile-overlay"
      v-if="mobileNav"
      @click="mobileNav = false"
    ></div>
    <div class="main-area">
      <header class="topbar">
        <button class="mobile-menu" @click="mobileNav = !mobileNav">
          <Menu :size="24" />
        </button>
        <div class="breadcrumb">
          <span>Башкирская языковая школа</span><span class="bread-sep">/</span
          ><strong>{{ nav.find((n) => n.id === tab)?.title }}</strong>
        </div>
        <div class="topbar-right">
          <span class="language-pill">🇧🇦 Башкирский → Русский</span
          ><span class="stat-chip flame"
            ><Flame :size="18" /> {{ progress?.streak || 0 }}</span
          ><span class="stat-chip xp"
            ><Star :size="18" /> {{ progress?.xp || 0 }} XP</span
          ><button class="avatar" @click="navigate('profile')">
            {{ user?.displayName?.charAt(0).toUpperCase() }}
          </button>
        </div>
      </header>
      <main class="content">
        <template v-if="tab === 'home'"
          ><div class="welcome-header">
            <div>
              <div class="eyebrow">ХӘЙЕРЛЕ КӨН! · ДОБРОГО ДНЯ!</div>
              <h1>
                С возвращением, {{ user?.displayName }}
                <span class="wave">👋</span>
              </h1>
              <p>
                Каждое выполненное задание помогает сохранить башкирский язык.
              </p>
            </div>
            <div class="today-tag">
              <span class="pulse-dot"></span> Ваш день изучения
            </div>
          </div>
          <div class="dashboard-grid">
            <div class="hero-panel">
              <div class="hero-panel-content">
                <span class="hero-label"
                  ><Sparkles :size="15" /> ВАШ ПУТЬ К ЗНАНИЯМ</span
                >
                <h2>Одно слово сегодня.<br />Тысяча завтра.</h2>
                <p>
                  Проверьте перевод, заработайте очки и помогите создать
                  открытую базу башкирского языка.
                </p>
                <button class="btn btn-white" @click="navigate('learn')">
                  Начать занятие <ArrowRight :size="18" />
                </button>
              </div>
              <div class="hero-illustration">
                <div class="book-back"></div>
                <div class="book-icon">Ә</div>
                <div class="spark s1">✦</div>
                <div class="spark s2">✧</div>
                <div class="spark s3">✦</div>
              </div>
            </div>
            <div class="level-panel card">
              <div class="section-label">
                <Crown :size="19" class="gold-text" /> ВАШ УРОВЕНЬ
              </div>
              <div class="level-number">
                {{ progress?.level || 1 }} <span>уровень</span>
              </div>
              <div class="level-description">Продолжайте в том же духе!</div>
              <div class="progress-track">
                <span
                  :style="{
                    width: ((progress?.xpInLevel || 0) / 200) * 100 + '%',
                  }"
                ></span>
              </div>
              <div class="progress-caption">
                <span>{{ progress?.xpInLevel || 0 }} / 200 XP</span
                ><span>Следующий уровень</span>
              </div>
              <div class="level-footer">
                <span>Всего заработано</span
                ><strong>{{ progress?.xp || 0 }} XP</strong>
              </div>
            </div>
          </div>
          <div class="stat-grid">
            <div class="metric-card">
              <span class="metric-icon violet"><ListChecks :size="24" /></span>
              <div>
                <strong>{{ progress?.completedTasks || 0 }}</strong
                ><span>Заданий выполнено</span>
              </div>
            </div>
            <div class="metric-card">
              <span class="metric-icon orange"><Flame :size="24" /></span>
              <div>
                <strong>{{ progress?.streak || 0 }} дней</strong
                ><span>Текущая серия</span>
              </div>
            </div>
            <div class="metric-card">
              <span class="metric-icon green"><CircleCheck :size="24" /></span>
              <div>
                <strong>{{ progress?.confirmedContributions || 0 }}</strong
                ><span>Подтверждённых вкладов</span>
              </div>
            </div>
          </div>
          <section class="card section-card home-quests">
            <div class="section-head">
              <h3>🎯 Ежедневные цели</h3>
              <button class="text-link" @click="navigate('learn')">
                К переводам <ArrowRight :size="15" />
              </button>
            </div>
            <div class="home-quest-list">
              <div
                v-for="q in dailyQuests"
                :key="q.code"
                class="home-quest-row"
              >
                <span>{{ q.rewarded ? "✅" : "🎯" }}</span>
                <div>
                  <strong>{{ q.title }}</strong>
                  <div class="quest-meter">
                    <div
                      :style="{ width: (q.progress / q.goal) * 100 + '%' }"
                    ></div>
                  </div>
                </div>
                <span
                  >{{ q.progress }}/{{ q.goal }} · +{{ q.rewardXp }} XP</span
                >
              </div>
            </div>
          </section>
          <div class="two-col">
            <section class="card section-card">
              <div class="section-head">
                <h3>Ваши достижения</h3>
                <button class="text-link" @click="navigate('profile')">
                  Все достижения <ArrowRight :size="15" />
                </button>
              </div>
              <div class="achievement-list">
                <div
                  v-for="a in achievements"
                  :key="a.code"
                  class="achievement"
                  :class="{ locked: !a.unlocked }"
                >
                  <span class="achievement-icon"><Medal :size="23" /></span>
                  <div>
                    <strong>{{ a.name }}</strong
                    ><span>{{ a.description }}</span>
                  </div>
                  <component
                    :is="a.unlocked ? Check : LockKeyhole"
                    :size="17"
                    :class="a.unlocked ? 'green-text' : 'muted'"
                  />
                </div>
              </div>
            </section>
            <section class="card section-card story-card">
              <div class="section-head">
                <h3>Зачем это важно?</h3>
                <BookHeart :size="21" class="green-text" />
              </div>
              <div class="story-quote">«Тел бар икән — халыҡ бар»</div>
              <p>
                Пока вы изучаете язык, каждый подтверждённый перевод становится
                частью общей базы, которая поможет будущим поколениям сохранить
                и понять башкирскую речь.
              </p>
              <div class="story-foot">
                <Heart :size="17" /> Сохраняем язык вместе
              </div>
            </section>
          </div>
        </template>
        <template v-else-if="tab === 'learn'"
          ><AudioClip
            v-if="task?.source?.hasAudio"
            :key="task.taskId"
            :source="task.source"
          />
          <section class="card section-card">
            <div class="form-row">
              <label class="pair-label"
                >Языковая пара<select v-model="chosenPair" @change="savePair">
                  <option
                    v-for="pair in languagePairs"
                    :key="pair.code"
                    :value="pair.code"
                  >
                    {{ pair.name }}
                  </option>
                </select></label
              >
            </div>
            <p class="subtext">
              Для каждого направления задания и голоса учитываются отдельно.
            </p>
          </section>
          <div class="page-heading">
            <div class="eyebrow">КАЖДЫЙ ДЕНЬ — НОВЫЕ ЦЕЛИ</div>
            <h1>Задания дня <span class="heading-emoji">🎯</span></h1>
            <p>
              Выполняйте переводы, закрывайте ежедневные цели и получайте
              бонусные XP!
            </p>
          </div>
          <section class="quest-grid" aria-label="Ежедневные цели">
            <article
              v-for="q in dailyQuests"
              :key="q.code"
              class="quest-card card"
              :class="{ 'quest-done': q.rewarded }"
            >
              <div class="quest-card-top">
                <span class="quest-symbol">{{
                  q.code === "one_correction"
                    ? "✍️"
                    : q.code === "five_translations"
                      ? "🏆"
                      : "📚"
                }}</span
                ><span class="quest-reward"
                  ><Star :size="15" /> +{{ q.rewardXp }} XP</span
                >
              </div>
              <h3>{{ q.title }}</h3>
              <p>{{ q.description }}</p>
              <div class="quest-meter">
                <div
                  :style="{ width: (q.progress / q.goal) * 100 + '%' }"
                ></div>
              </div>
              <div class="quest-bottom">
                <strong>{{ q.progress }} / {{ q.goal }}</strong
                ><span v-if="q.rewarded" class="quest-success"
                  ><Check :size="15" /> Получено</span
                ><span v-else>До награды</span>
              </div>
            </article>
          </section>
          <div class="lesson-wrap">
            <div class="lesson-card card" v-if="task">
              <div class="lesson-top">
                <span class="pill pill-green"
                  ><BookOpen :size="15" /> Перевод</span
                ><span class="lesson-reward"><Star :size="17" /> +10 XP</span>
              </div>
              <div v-if="!completedNow">
                <div class="lesson-prompt">
                  Прочитайте предложение на исходном языке
                </div>
                <div class="bashkir-text">{{ task.sourceText }}</div>
                <div class="translation-label">ПРЕДЛОЖЕННЫЙ ПЕРЕВОД</div>
                <div class="suggested-box">
                  <span class="translation-mark">Аа</span
                  >{{ task.suggestedTranslation }}
                </div>
                <div class="lesson-question">Этот перевод правильный?</div>
                <template v-if="!correcting"
                  ><div class="lesson-actions">
                    <button
                      class="btn btn-primary"
                      :disabled="loading"
                      @click="submitAnswer('confirmation')"
                    >
                      <Check :size="19" /> Да, подтвердить</button
                    ><button class="btn btn-outline" @click="correcting = true">
                      <PenLine :size="19" /> Исправить
                    </button>
                  </div></template
                ><template v-else
                  ><label class="form-label">Напишите ваш вариант</label
                  ><textarea
                    v-model="correction"
                    rows="3"
                    placeholder="Введите правильный перевод..."
                    maxlength="2000"
                  ></textarea>
                  <div class="lesson-actions">
                    <button
                      class="btn btn-primary"
                      :disabled="loading || !correction.trim()"
                      @click="submitAnswer('correction')"
                    >
                      <Send :size="18" /> Отправить исправление</button
                    ><button class="btn btn-ghost" @click="correcting = false">
                      Назад
                    </button>
                  </div></template
                >
              </div>
              <div v-else class="completed-state">
                <div class="completed-badge"><Check :size="42" /></div>
                <h2>Афарин! Отлично!</h2>
                <p>
                  Ваш ответ сохранён. Благодаря вам база башкирских переводов
                  становится лучше.
                </p>
                <div class="earned">
                  <Star :size="22" /> +{{ lastAnswer?.xpEarned || 10 }} XP
                </div>
                <div v-if="lastAnswer?.questBonusXp" class="quest-celebrate">
                  🎉 Выполнена ежедневная цель! Бонус: +{{
                    lastAnswer.questBonusXp
                  }}
                  XP
                </div>
                <div class="note">
                  Новые переводы подтверждаются после согласования ответов
                  нескольких участников.
                </div>
                <button class="btn btn-primary" @click="safe(loadTask)">
                  Следующий перевод <ArrowRight :size="18" />
                </button>
              </div>
            </div>
            <div v-else class="card empty-state">
              <BookOpen :size="40" />
              <h3>Загружаем упражнение</h3>
              <button class="btn btn-primary" @click="safe(loadTask)">
                Обновить
              </button>
            </div>
            <div class="lesson-side">
              <div class="tip-card">
                <div class="tip-icon">💡</div>
                <h3>Как это работает?</h3>
                <p>
                  Подтвердите перевод, если он верный, или напишите свой
                  вариант. Ваш ответ поможет собрать качественную базу
                  переводов.
                </p>
              </div>
              <div class="mini-card">
                <Flame :size="22" />
                <div>
                  <strong>{{ progress?.streak || 0 }} дней</strong
                  ><span>Ваша серия занятий</span>
                </div>
              </div>
            </div>
          </div></template
        >
        <template v-else-if="tab === 'leaderboard'"
          ><div class="page-heading">
            <div class="eyebrow">СОРЕВНУЙТЕСЬ И ВДОХНОВЛЯЙТЕСЬ</div>
            <h1>Таблица лидеров 🏆</h1>
            <p v-if="boardMetric === 'vocabulary'">
              Число уникальных слов в фрагментах с правильным ответом по
              проверенному переводу.
            </p>
            <p>Соревнуйтесь с другими участниками, классами и школами.</p>
          </div>
          <section class="card leaderboard-card">
            <div class="filters">
              <label
                >Участники<select
                  v-model="boardScope"
                  @change="safe(loadLeaders)"
                >
                  <option value="global">Все ученики</option>
                  <option value="school">Школы</option>
                  <option value="class">Классы</option>
                </select></label
              ><label
                >Показатель<select
                  v-model="boardMetric"
                  @change="safe(loadLeaders)"
                >
                  <option value="xp">Опыт (XP)</option>
                  <option value="activity">Выполненные задания</option>
                  <option value="vocabulary">Словарный запас</option>
                  <option value="corrections">Исправления</option>
                  <option value="errors">Меньше ошибок</option>
                </select></label
              ><label
                >Период<select
                  v-model="boardPeriod"
                  @change="safe(loadLeaders)"
                >
                  <option value="daily">Сегодня</option>
                  <option value="weekly">Эта неделя</option>
                  <option value="monthly">Этот месяц</option>
                  <option value="yearly">Этот год</option>
                  <option value="all">За всё время</option>
                </select></label
              ><button class="icon-btn" @click="safe(loadLeaders)">
                <RefreshCw :size="18" />
              </button>
            </div>
            <div v-if="leaders.length" class="leader-list">
              <div
                v-for="entry in leaders"
                :key="entry.entityId"
                class="leader-row"
                :class="{ 'self-row': entry.entityId === user?.id }"
              >
                <div class="rank" :class="{ 'top-rank': entry.rank < 4 }">
                  {{
                    entry.rank === 1
                      ? "🥇"
                      : entry.rank === 2
                        ? "🥈"
                        : entry.rank === 3
                          ? "🥉"
                          : entry.rank
                  }}
                </div>
                <div class="leader-avatar">
                  {{ entry.displayName.charAt(0) }}
                </div>
                <div class="leader-name">
                  {{ entry.displayName }}
                  <span v-if="entry.entityId === user?.id">(вы)</span>
                </div>
                <div class="leader-score">
                  <Star :size="16" /> {{ entry.score }}
                  {{
                    boardMetric === "xp"
                      ? "XP"
                      : boardMetric === "vocabulary"
                        ? "слов"
                        : ""
                  }}
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <Trophy :size="44" />
              <h3>Рейтинг пока пуст</h3>
              <p>Выполните первое упражнение и появитесь в таблице.</p>
            </div>
          </section></template
        >
        <template v-else-if="tab === 'library'"
          ><div class="page-heading">
            <div class="eyebrow">ОБЩЕЕ НАСЛЕДИЕ</div>
            <h1>Библиотека переводов 📚</h1>
            <p>Подтверждённые переводы, собранные участниками платформы.</p>
          </div>
          <div class="section-head">
            <h3>
              Проверенные переводы
              <span class="small-count">{{ translations.length }}</span>
            </h3>
            <div class="inline-actions">
              <button
                class="btn btn-small btn-outline"
                @click="exportData('csv')"
              >
                <Download :size="17" /> CSV</button
              ><button
                class="btn btn-small btn-outline"
                @click="exportData('json')"
              >
                <Download :size="17" /> JSON
              </button>
            </div>
          </div>
          <div v-if="translations.length" class="translation-grid">
            <div
              v-for="item in translations"
              :key="item.id"
              class="card translation-card"
            >
              <div class="translation-card-top">
                <span class="pill pill-green"
                  ><Check :size="13" /> Подтверждено</span
                ><span class="muted">{{ item.source.title }}</span>
              </div>
              <strong>{{ item.textBa }}</strong>
              <div class="translation-divider"></div>
              <p>{{ item.translation }}</p>
              <div class="translation-card-footer">
                <Users :size="15" /> {{ item.confirmed }} подтверждений
              </div>
            </div>
          </div>
          <div v-else class="card empty-state">
            <Library :size="47" />
            <h3>Первые переводы ещё впереди</h3>
            <p>
              Перевод появится здесь, когда минимум 5 разных участников
              достигнут согласия.
            </p>
            <button class="btn btn-primary" @click="navigate('learn')">
              Помочь с переводами
            </button>
          </div>
          <section class="card section-card library-source">
            <h3>Источники материалов</h3>
            <div
              v-for="material in materials"
              :key="material.id"
              class="material-row"
            >
              <BookOpen :size="21" />
              <div>
                <strong>{{ material.title }}</strong
                ><span>{{ material.description || "Учебный материал" }}</span>
              </div>
              <span class="pill">{{ material.type }}</span>
            </div>
          </section></template
        >
        <template v-else-if="tab === 'schools'"
          ><div class="page-heading">
            <div class="eyebrow">УЧИМСЯ ВМЕСТЕ</div>
            <h1>Школы и классы 🏫</h1>
            <p>
              Объединяем учеников и преподавателей вокруг башкирского языка.
            </p>
          </div>
          <section class="card section-card">
            <div class="section-head">
              <h3>Учебные заведения</h3>
              <form
                v-if="user?.role === 'admin'"
                @submit.prevent="createSchool"
              >
                <input
                  v-model="schoolName"
                  required
                  placeholder="Название школы"
                /><button class="btn btn-outline">Добавить школу</button>
              </form>
              <GraduationCap :size="23" class="green-text" />
            </div>
            <div class="schools-grid">
              <div v-for="s in schools" :key="s.id" class="school-card">
                <div class="school-icon"><School :size="27" /></div>
                <strong>{{ s.name }}</strong
                ><span>Участвует в языковой школе</span>
              </div>
            </div>
          </section>
          <section v-if="isTeacher" class="card section-card">
            <h3>Управление классами</h3>
            <div class="form-row">
              <select v-model="selectedSchool">
                <option value="">Выберите школу</option>
                <option v-for="s in schools" :value="s.id" :key="s.id">
                  {{ s.name }}
                </option></select
              ><input
                v-model="newClassName"
                placeholder="Название класса"
              /><button class="btn btn-primary" @click="createClass">
                <Plus :size="17" /> Создать
              </button>
            </div>
            <div v-for="cl in classes" :key="cl.id" class="class-row">
              <div>
                <strong>{{ cl.name }}</strong
                ><span>{{ cl.school }}</span>
              </div>
              <button
                class="btn btn-small btn-outline"
                @click="showClass(cl.id)"
              >
                Отчёт
              </button>
            </div>
            <div v-if="classStats" class="class-details">
              <h4>Статистика класса</h4>
              <form class="stack-form" @submit.prevent="registerClassStudent">
                <h4>Зарегистрировать ученика</h4>
                <input
                  v-model="studentRegistration.displayName"
                  required
                  minlength="2"
                  placeholder="Имя ученика"
                /><input
                  v-model="studentRegistration.email"
                  type="email"
                  autocomplete="email"
                  required
                  placeholder="Почта ученика"
                /><input
                  v-model="studentRegistration.password"
                  type="password"
                  required
                  minlength="8"
                  maxlength="72"
                  placeholder="Начальный пароль"
                /><button class="btn btn-outline">
                  Создать аккаунт в классе
                </button>
              </form>
              <p>
                Учеников: {{ classStats.studentsCount }} · Активных:
                {{ classStats.activeStudents }} · Выполненных заданий:
                {{ classStats.totalCompletedTasks }}
              </p>
              <div v-for="s in classStudents" :key="s.userId" class="class-row">
                <span>{{ s.displayName }}</span
                ><strong>{{ s.totalXp }} XP</strong>
              </div>
              <div class="form-row">
                <input
                  v-model="studentToAdd"
                  placeholder="UUID ученика"
                /><button class="btn btn-small btn-primary" @click="addStudent">
                  Добавить
                </button>
              </div>
            </div>
          </section>
          <div v-else class="card info-box">
            <Users :size="22" />
            <p>
              Создавать классы и просматривать успеваемость могут преподаватели.
              Роль преподавателя назначает администратор.
            </p>
          </div></template
        >
        <template v-else-if="tab === 'profile'"
          ><div class="page-heading">
            <div class="eyebrow">ЛИЧНЫЙ КАБИНЕТ</div>
            <h1>Мой профиль 👤</h1>
            <p>Всё о вашем языковом путешествии.</p>
          </div>
          <div class="profile-hero card">
            <div class="profile-avatar">
              {{ user?.displayName.charAt(0).toUpperCase() }}
            </div>
            <div class="profile-details">
              <h2>{{ user?.displayName }}</h2>
              <p>{{ user?.email }}</p>
              <p class="uid-line">ID: {{ user?.id }}</p>
              <span class="pill pill-green">{{
                user?.role === "student"
                  ? "Ученик"
                  : user?.role === "teacher"
                    ? "Преподаватель"
                    : user?.role === "admin"
                      ? "Администратор"
                      : "Модератор"
              }}</span>
            </div>
            <div class="profile-level">
              <Crown :size="27" /><strong
                >Уровень {{ progress?.level || 1 }}</strong
              >
            </div>
          </div>
          <div class="stat-grid">
            <div class="metric-card">
              <span class="metric-icon violet"><Star :size="24" /></span>
              <div>
                <strong>{{ progress?.xp || 0 }}</strong
                ><span>Всего XP</span>
              </div>
            </div>
            <div class="metric-card">
              <span class="metric-icon orange"><Flame :size="24" /></span>
              <div>
                <strong>{{ progress?.longestStreak || 0 }}</strong
                ><span>Лучшая серия</span>
              </div>
            </div>
            <div class="metric-card">
              <span class="metric-icon green"><ListChecks :size="24" /></span>
              <div>
                <strong>{{ progress?.completedTasks || 0 }}</strong
                ><span>Заданий выполнено</span>
              </div>
            </div>
          </div>
          <section class="card section-card">
            <h3>Достижения</h3>
            <div class="profile-achievements">
              <div
                v-for="a in achievements"
                :key="a.code"
                class="achievement-item"
                :class="{ locked: !a.unlocked }"
              >
                <div class="award-emoji"><Medal :size="29" /></div>
                <strong>{{ a.name }}</strong>
                <p>{{ a.description }}</p>
                <span>{{ a.unlocked ? "Получено" : "Не открыто" }}</span>
              </div>
            </div>
          </section>
          <section class="card section-card">
            <h3>Моя языковая пара</h3>
            <div class="form-row">
              <select v-model="chosenPair">
                <option
                  v-for="pair in languagePairs"
                  :key="pair.code"
                  :value="pair.code"
                >
                  {{ pair.name }}
                </option></select
              ><button class="btn btn-primary" @click="savePair">
                Сохранить
              </button>
            </div>
          </section>
          <section class="card section-card">
            <h3>Моя школа</h3>
            <div class="form-row">
              <select v-model="profileSchool">
                <option value="">Без школы</option>
                <option v-for="s in schools" :key="s.id" :value="s.id">
                  {{ s.name }}
                </option></select
              ><button class="btn btn-primary" @click="updateSchool">
                Сохранить
              </button>
            </div>
          </section>
          <section class="card section-card">
            <div class="section-head">
              <h3>Telegram-бот</h3>
              <Send :size="21" class="green-text" />
            </div>
            <p class="subtext">
              Привяжите Telegram, чтобы получать ежедневные задания прямо в
              мессенджере.
            </p>
            <button class="btn btn-outline" @click="getLink">
              Получить код привязки
            </button>
            <div v-if="linkCode" class="code-box">
              Отправьте боту: <code>/start {{ linkCode }}</code>
              <p>Код действителен 15 минут. Не передавайте его другим людям.</p>
            </div>
          </section></template
        >
        <template v-else-if="tab === 'moderation' && isStaff"
          ><div class="page-heading">
            <div class="eyebrow">УПРАВЛЕНИЕ КОНТЕНТОМ</div>
            <h1>Панель модератора 🛡️</h1>
            <p>Работа со спорными переводами и учебными материалами.</p>
          </div>
          <section class="card section-card">
            <div class="section-head">
              <h3>Спорные переводы</h3>
              <span class="pill"
                >{{
                  cases.filter((x) => x.status === "open").length
                }}
                открытых</span
              >
            </div>
            <div v-if="!cases.length" class="empty-state compact">
              <ShieldCheck :size="38" />
              <h3>Нет спорных переводов</h3>
              <p>Они появятся, когда участники не смогут прийти к согласию.</p>
            </div>
            <div v-for="mc in cases" :key="mc.id" class="case-row">
              <div>
                <strong>{{ mc.textBa }}</strong
                ><span>{{ mc.langPair }} · {{ mc.status }}</span>
              </div>
              <button
                v-if="mc.status === 'open'"
                class="btn btn-small btn-outline"
                @click="loadCandidates(mc.id)"
              >
                Рассмотреть
              </button>
            </div>
            <div v-if="candidates.length" class="review-box">
              <h4>Выберите подтверждённый вариант</h4>
              <label v-for="ca in candidates" :key="ca.id" class="candidate"
                ><input
                  v-model="resolveCandidate"
                  type="radio"
                  :value="ca.id"
                /><span>{{ ca.translation }}</span
                ><small>{{ ca.votes }} голосов</small></label
              >
              <div class="form-row">
                <select v-model="aliasCandidate">
                  <option value="">Синонимичный вариант</option>
                  <option v-for="ca in candidates" :key="ca.id" :value="ca.id">
                    {{ ca.translation }}
                  </option></select
                ><select v-model="aliasCanonical">
                  <option value="">Объединить с вариантом</option>
                  <option v-for="ca in candidates" :key="ca.id" :value="ca.id">
                    {{ ca.translation }}
                  </option></select
                ><button class="btn btn-outline" @click="mergeSynonyms">
                  Объединить синонимы
                </button>
              </div>
              <button class="btn btn-primary" @click="resolve">
                Утвердить перевод
              </button>
            </div>
          </section>
          <div v-if="user?.role === 'admin'" class="two-col">
            <section class="card section-card">
              <h3>Добавить материал</h3>
              <div class="stack-form">
                <input
                  v-model="newMaterial.title"
                  placeholder="Название источника"
                /><select v-model="newMaterial.type">
                  <option value="text">Текст</option>
                  <option value="audio">Аудио</option></select
                ><input
                  v-model="newMaterial.license"
                  placeholder="Лицензия / разрешение"
                /><textarea
                  v-model="newMaterial.description"
                  placeholder="Описание источника"
                  rows="3"
                ></textarea
                ><button class="btn btn-primary" @click="createMaterial">
                  Создать материал
                </button>
              </div>
            </section>
            <section class="card section-card">
              <h3>Добавить фрагмент</h3>
              <div class="stack-form">
                <select v-model="newFragment.materialId">
                  <option value="">Выберите материал</option>
                  <option v-for="m in materials" :key="m.id" :value="m.id">
                    {{ m.title }}
                  </option></select
                ><input
                  v-model="newFragment.sourceLang"
                  maxlength="3"
                  placeholder="Код языка исходного текста (ba, ru, en...)"
                /><textarea
                  v-model="newFragment.textBa"
                  placeholder="Исходный текст фрагмента"
                  rows="3"
                ></textarea
                ><input
                  v-model="newFragment.chapter"
                  placeholder="Глава или раздел"
                /><button class="btn btn-primary" @click="createFragment">
                  Сохранить фрагмент</button
                ><template v-if="createdFragmentId"
                  ><div class="translation-label">
                    ИСХОДНЫЙ ВАРИАНТ ПЕРЕВОДА
                  </div>
                  <select v-model="seedPair">
                    <option
                      v-for="pair in languagePairs.filter(
                        (p) => p.sourceLang === createdFragmentSource,
                      )"
                      :key="pair.code"
                      :value="pair.code"
                    >
                      {{ pair.name }}
                    </option></select
                  ><input
                    v-model="seedTranslation"
                    placeholder="Предложенный перевод на выбранный язык"
                  /><button
                    class="btn btn-outline"
                    @click="createSeedTranslation"
                  >
                    Добавить перевод
                  </button></template
                >
              </div>
            </section>
          </div>
          <AudioImport
            v-if="user?.role === 'admin'"
            @imported="loadLiteratureDrafts"
          />
          <section
            v-if="user?.role === 'admin'"
            class="card section-card literature-admin"
          >
            <h3>📖 Загрузка художественных произведений</h3>
            <p class="subtext">
              Доступно только администратору. Загружайте TXT в UTF-8 или
              вставьте текст, подтверждая право использования. Система
              автоматически нарежет текст на предложения, а ИИ предложит
              переводы. До проверки материалы не появятся в заданиях.
            </p>
            <div class="form-row">
              <input
                v-model="literatureMeta.title"
                placeholder="Название произведения"
                maxlength="200"
              /><input
                v-model="literatureMeta.author"
                placeholder="Автор"
                maxlength="200"
              />
            </div>
            <div class="form-row">
              <label
                >Язык оригинала
                <input
                  v-model="literatureMeta.sourceLang"
                  maxlength="3"
                  placeholder="ba" /></label
              ><label
                >Лицензия / основание использования
                <input
                  v-model="literatureMeta.license"
                  placeholder="public-domain или разрешение правообладателя"
              /></label>
            </div>
            <textarea
              v-model="literatureMeta.copyrightNote"
              rows="2"
              placeholder="Информация о правах и источнике (необязательно)"
              maxlength="500"
            ></textarea>
            <div class="form-row">
              <label
                >Тип фрагмента
                <select v-model="literatureSegmentMode">
                  <option value="sentence">Предложения</option>
                  <option value="paragraph">Абзацы</option>
                </select></label
              >
            </div>
            <div class="form-row">
              <label class="rights-label"
                ><input v-model="literatureRightsConfirmed" type="checkbox" />
                Подтверждаю, что имею право использовать и обрабатывать этот
                текст, включая передачу фрагментов внешнему ИИ-сервису.</label
              >
            </div>
            <div class="form-row">
              <label
                >TXT-файл (UTF-8, до 1 МБ)
                <input
                  type="file"
                  accept=".txt,text/plain"
                  @change="onLiteratureFile"
              /></label>
            </div>
            <textarea
              v-if="!literatureFile"
              v-model="literatureText"
              rows="5"
              placeholder="Или вставьте полный текст произведения..."
            ></textarea>
            <button
              class="btn btn-primary"
              :disabled="
                loading ||
                !literatureMeta.title ||
                (!literatureFile && !literatureText.trim()) ||
                !literatureRightsConfirmed
              "
              @click="uploadLiterature"
            >
              <Plus :size="17" /> Импортировать и нарезать текст
            </button>
            <div class="section-head" style="margin-top: 25px">
              <h3>Черновики на проверке ({{ literatureDrafts.length }})</h3>
              <button
                class="btn btn-small btn-outline"
                @click="safe(loadLiteratureDrafts)"
              >
                <RefreshCw :size="16" /> Обновить
              </button>
            </div>
            <div class="form-row">
              <label
                >Направление для генерации
                <select
                  v-model="literaturePair"
                  @change="safe(loadLiteratureDrafts)"
                >
                  <option
                    v-for="pair in languagePairs"
                    :key="pair.code"
                    :value="pair.code"
                  >
                    {{ pair.name }}
                  </option>
                </select></label
              >
            </div>
            <p class="subtext">
              Для ИИ требуется AI_API_KEY в .env сервера. Перевод и намеренная
              ошибка — только предложения модели; администратор обязан проверить
              обе версии. Без ключа импорт и ручная работа с фрагментами
              доступны.
            </p>
            <div
              v-for="fragment in literatureDrafts"
              :key="fragment.id"
              class="literature-draft"
            >
              <div>
                <strong>{{ fragment.text }}</strong>
                <p class="subtext">
                  {{ fragment.title }} · {{ fragment.chapter }} · язык
                  {{ fragment.sourceLang }} · вариантов:
                  {{ fragment.candidates }}
                </p>
              </div>
              <div
                v-if="literatureAIResults[fragment.id] || fragment.correct"
                class="literature-ai-result"
              >
                <p>
                  <b>Предположительно верно:</b>
                  {{
                    literatureAIResults[fragment.id]?.correct ||
                    fragment.correct
                  }}
                </p>
                <p>
                  <b>Преднамеренная ошибка:</b>
                  {{
                    literatureAIResults[fragment.id]?.incorrect ||
                    fragment.incorrect
                  }}
                </p>
                <p class="subtext">
                  {{
                    literatureAIResults[fragment.id]?.errorExplanation ||
                    fragment.errorExplanation
                  }}
                </p>
              </div>
              <div class="inline-actions">
                <button
                  class="btn btn-small btn-outline"
                  :disabled="!!literatureBusy"
                  @click="generateLiterature(fragment.id)"
                >
                  ✨ Сгенерировать ИИ
                </button>
                <div v-if="literatureAIResults[fragment.id]" class="stack-form">
                  <label
                    >Правильный перевод<textarea
                      v-model="literatureAIResults[fragment.id]!.correct"
                      maxlength="2000"
                    /></label
                  ><label
                    >Заведомо ошибочный перевод<textarea
                      v-model="literatureAIResults[fragment.id]!.incorrect"
                      maxlength="2000"
                    /></label
                  ><button
                    class="btn btn-outline"
                    @click="saveLiteratureTranslations(fragment.id)"
                  >
                    Сохранить проверенные варианты
                  </button>
                </div>
                <button
                  class="btn btn-small btn-primary"
                  :disabled="!!literatureBusy"
                  @click="moderateLiterature(fragment.id, 'publish')"
                >
                  Опубликовать
                </button>
                <button
                  class="btn btn-small btn-outline"
                  :disabled="!!literatureBusy"
                  @click="moderateLiterature(fragment.id, 'reject')"
                >
                  Отклонить
                </button>
              </div>
            </div>
            <p v-if="!literatureDrafts.length" class="subtext">
              Нет черновиков. Добавьте произведение для начала работы.
            </p>
          </section>
          <section v-if="user?.role === 'admin'" class="card section-card">
            <h3>Управление языковыми парами</h3>
            <p class="subtext">
              Добавляйте направления перевода без изменения Go-кода. Чтобы
              задания работали, добавьте исходные фрагменты и варианты перевода
              на выбранный язык.
            </p>
            <div class="form-row">
              <input
                v-model="newPair.sourceLang"
                maxlength="3"
                placeholder="Код исходного языка (ba)"
              /><input
                v-model="newPair.targetLang"
                maxlength="3"
                placeholder="Код языка перевода (en)"
              /><input
                v-model="newPair.name"
                placeholder="Название (Башкирский → Английский)"
              /><button class="btn btn-primary" @click="createLanguagePair">
                <Plus :size="17" /> Добавить
              </button>
            </div>
            <div
              v-for="pair in allLanguagePairs"
              :key="pair.code"
              class="class-row"
            >
              <div>
                <strong>{{ pair.name }}</strong
                ><span
                  >{{ pair.code }} ·
                  {{ pair.active ? "Активна" : "Отключена" }}</span
                >
              </div>
              <div class="inline-actions">
                <button
                  class="btn btn-small btn-outline"
                  @click="updatePairName(pair)"
                >
                  Переименовать</button
                ><button
                  class="btn btn-small btn-outline"
                  @click="togglePair(pair)"
                >
                  {{ pair.active ? "Отключить" : "Включить" }}
                </button>
              </div>
            </div>
          </section>
          <section v-if="user?.role === 'admin'" class="card section-card">
            <h3>Роли пользователей</h3>
            <p class="subtext">
              Назначьте роль по UUID из профиля пользователя.
            </p>
            <div class="form-row">
              <input
                v-model="newRoleUserId"
                placeholder="UUID пользователя"
              /><select v-model="newRole">
                <option value="student">Ученик</option>
                <option value="teacher">Преподаватель</option>
                <option value="moderator">Модератор</option>
                <option value="admin">Администратор</option></select
              ><button class="btn btn-primary" @click="changeRole">
                Назначить
              </button>
            </div>
          </section></template
        >
      </main>
      <footer class="footer">
        © 2026 ТелемКүпер · Башкирская языковая школа
        <span>Сохраняем язык вместе 💚</span>
      </footer>
    </div>
    <div v-if="toast" class="toast">
      <Sparkles :size="20" /> {{ toast }}
      <button @click="toast = ''" aria-label="Закрыть"><X :size="16" /></button>
    </div>
  </div>
</template>
