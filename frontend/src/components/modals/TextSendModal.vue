<script setup lang="ts">
// --- Vue 核心 ---
import { computed, ref, watch, nextTick } from "vue";
import { useI18n } from "vue-i18n";

// --- Wails & 后端绑定 ---
import { SendText } from "../../../bindings/mesh-drop/internal/transfer/service";
import { Peer } from "../../../bindings/mesh-drop/internal/discovery/models";

// --- 属性 & 事件 ---
const props = defineProps<{
  modelValue: boolean;
  peer: Peer;
  selectedIp: string;
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void;
  (e: "transferStarted"): void;
}>();

// --- 状态 ---
const { t } = useI18n();
const textContent = ref("");
const textareaRef = ref();

// --- 计算属性 ---
const show = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

// --- 监听 ---
watch(show, async (val) => {
  if (val) {
    await nextTick();
    textareaRef.value?.focus();
  }
});

// --- 方法 ---
const executeSendText = async () => {
  if (!props.selectedIp || !textContent.value) return;

  try {
    await SendText(props.peer, props.selectedIp, textContent.value);
    emit("transferStarted");
    show.value = false;
    textContent.value = "";
  } catch (e) {
    console.error(e);
    alert(t("modal.textSend.failed", { error: e }));
  }
};
</script>

<template>
  <v-dialog v-model="show" width="500" persistent eager>
    <v-card :title="$t('modal.textSend.title')">
      <v-card-text>
        <v-textarea
          ref="textareaRef"
          v-model="textContent"
          :label="$t('modal.textSend.contentLabel')"
          :placeholder="$t('modal.textSend.placeholder')"
          rows="4"
          auto-grow
        ></v-textarea>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="show = false">{{
          $t("common.cancel")
        }}</v-btn>
        <v-btn
          color="primary"
          @click="executeSendText"
          :disabled="!textContent"
        >
          {{ $t("modal.textSend.send") }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
