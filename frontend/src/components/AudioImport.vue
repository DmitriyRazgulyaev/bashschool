<script setup lang="ts">
import { ref } from "vue";
import { API } from "../api";
const emit = defineEmits<{ imported: [] }>();
const title = ref(""),
  license = ref(""),
  transcript = ref(""),
  rights = ref(false),
  file = ref<File | null>(null),
  busy = ref(false),
  message = ref("");
async function submit() {
  busy.value = true;
  message.value = "";
  try {
    const body = new FormData();
    for (const [key, value] of Object.entries({
      title: title.value,
      license: license.value,
      sourceLang: "ba",
      transcript: transcript.value,
      rightsConfirmed: String(rights.value),
    }))
      body.append(key, value);
    if (file.value) body.append("audio", file.value);
    const response = await fetch(`${API}/admin/audio/import`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${localStorage.getItem("bashkir_token") || ""}`,
      },
      body,
    });
    const result = await response.json();
    if (!response.ok)
      throw new Error(result.error?.message || "Не удалось загрузить аудио");
    message.value = `Создано отрезков: ${result.fragmentsCreated}. Подготовьте переводы в списке черновиков.`;
    emit("imported");
  } catch (e) {
    message.value = e instanceof Error ? e.message : "Ошибка загрузки";
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <section class="card section-card">
    <h3>Аудиоматериалы</h3>
    <form class="stack-form" @submit.prevent="submit">
      <label>Название<input v-model="title" required maxlength="200" /></label
      ><label
        >Лицензия или разрешение<input
          v-model="license"
          required
          minlength="3" /></label
      ><label
        >Аудио MP3, WAV или OGG (до 20 МБ)<input
          type="file"
          accept="audio/mpeg,audio/wav,audio/ogg,.ogg"
          required
          @change="
            file = ($event.target as HTMLInputElement).files?.[0] || null
          " /></label
      ><label
        >Расшифровка WebVTT с таймкодами<textarea
          v-model="transcript"
          required
          rows="8"
          placeholder="WEBVTT&#10;&#10;00:00.000 --> 00:03.000&#10;Һаумыһығыҙ!"
        /></label
      ><label
        ><input v-model="rights" type="checkbox" required /> Есть право
        использовать аудио и расшифровку</label
      ><button class="btn btn-primary" :disabled="busy">
        {{ busy ? "Загрузка…" : "Создать отрезки" }}
      </button>
      <p v-if="message" role="status">{{ message }}</p>
    </form>
  </section>
</template>
