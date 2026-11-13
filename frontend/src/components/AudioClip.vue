<script setup lang="ts">
import { ref, onBeforeUnmount } from "vue";
import { API, type Task } from "../api";
const props = defineProps<{ source: NonNullable<Task["source"]> }>();
const audio = ref<HTMLAudioElement | null>(null);
const url = ref("");
const error = ref("");
const loading = ref(false);
let controller: AbortController | null = null;
async function load() {
  loading.value = true;
  error.value = "";
  controller = new AbortController();
  try {
    const response = await fetch(
      `${API}/materials/${props.source.materialId}/audio`,
      {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("bashkir_token") || ""}`,
        },
        signal: controller.signal,
      },
    );
    if (!response.ok) throw new Error("Не удалось загрузить аудио");
    url.value = URL.createObjectURL(await response.blob());
  } catch (e) {
    if (e instanceof Error && e.name !== "AbortError") error.value = e.message;
  } finally {
    loading.value = false;
  }
}
function seek() {
  if (audio.value)
    audio.value.currentTime = (props.source.startTimeMs || 0) / 1000;
}
function bound() {
  if (
    audio.value &&
    props.source.endTimeMs != null &&
    audio.value.currentTime >= props.source.endTimeMs / 1000
  ) {
    audio.value.pause();
    seek();
  }
}
onBeforeUnmount(() => {
  controller?.abort();
  if (url.value) URL.revokeObjectURL(url.value);
});
</script>
<template>
  <div class="audio-clip">
    <p>{{ source.title }} · {{ source.chapter }}</p>
    <button
      v-if="!url"
      class="btn btn-outline"
      :disabled="loading"
      @click="load"
    >
      {{ loading ? "Загрузка…" : "Слушать фрагмент" }}</button
    ><audio
      v-else
      ref="audio"
      :src="url"
      controls
      @loadedmetadata="seek"
      @timeupdate="bound"
    />
    <p v-if="error" role="alert">{{ error }}</p>
  </div>
</template>
