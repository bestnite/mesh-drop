<script setup lang="ts">
// --- Vue 核心 ---
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

// --- Wails & 后端绑定 ---
import { Events, Dialogs, Window } from "@wailsio/runtime";
import { SendFiles } from "../../../bindings/mesh-drop/internal/transfer/service";
import { Peer } from "../../../bindings/mesh-drop/internal/discovery/models";
import { File } from "bindings/mesh-drop/models";

onMounted(() => {});

// --- 属性 & 事件 ---
const props = defineProps<{
  modelValue: boolean;
  peer: Peer;
  selectedIp: string;
  files: File[];
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: boolean): void;
  (e: "transferStarted"): void;
}>();

// --- 状态 ---
const { t } = useI18n();

// --- 计算属性 ---
const show = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

// --- 监听 ---
watch(show, (newVal) => {
  if (newVal) {
    Events.On("files-dropped", (event) => {
      const files: File[] = event.data.files || [];
      files.forEach((f) => {
        if (!props.files.find((existing) => existing.path === f.path)) {
          props.files.push(f);
        }
      });
    });
  } else {
    Events.Off("files-dropped");
  }
});

// --- 方法 ---
const openFileDialog = async () => {
  const files = await Dialogs.OpenFile({
    Title: t("modal.fileSend.selectTitle"),
    AllowsMultipleSelection: true,
  });

  if (files) {
    if (Array.isArray(files)) {
      files.forEach((f) => {
        if (!props.files.find((existing) => existing.path === f)) {
          props.files.push({
            name: f.split(/[\/]/).pop() || f,
            path: f,
          });
        }
      });
    } else {
      const f = files as string;
      if (!props.files.find((existing) => existing.path === f)) {
        props.files.push({
          name: f.split(/[\\/]/).pop() || f,
          path: f,
        });
      }
    }
  }
};

const handleRemoveFile = (index: number) => {
  props.files.splice(index, 1);
};

const handleSendFiles = async () => {
  if (props.files.length === 0 || !props.selectedIp) return;
  const paths = props.files.map((f) => f.path);

  try {
    await SendFiles(props.peer, props.selectedIp, paths);
    emit("transferStarted");
    show.value = false;
  } catch (e) {
    console.error(e);
    alert(t("modal.fileSend.failed", { error: e }));
  }
};
</script>

<template>
  <v-dialog v-model="show" width="600" persistent eager>
    <v-card :title="$t('modal.fileSend.title')">
      <v-card-text>
        <div
          v-if="props.files.length === 0"
          class="drop-zone pa-10 text-center rounded-lg"
          @click="openFileDialog"
          data-file-drop-target
          id="drop-zone-area"
        >
          <v-icon
            icon="mdi-cloud-upload"
            size="48"
            color="primary"
            class="mb-2"
          ></v-icon>
          <div class="text-body-1 text-medium-emphasis">
            {{ $t("modal.fileSend.dragDrop") }}
          </div>
        </div>

        <div v-else>
          <v-list
            class="mb-4 text-left"
            border
            rounded
            max-height="400"
            style="overflow-y: auto"
            data-file-drop-target
            id="drop-zone-list"
          >
            <v-list-item
              v-for="(file, index) in props.files"
              :key="file.path"
              :title="file.name"
              :subtitle="file.path"
              lines="two"
            >
              <template #append>
                <v-btn
                  icon="mdi-delete"
                  size="small"
                  variant="text"
                  color="error"
                  @click="handleRemoveFile(index)"
                ></v-btn>
              </template>
            </v-list-item>
          </v-list>

          <v-btn
            block
            variant="tonal"
            prepend-icon="mdi-plus"
            @click="openFileDialog"
            class="mt-2"
          >
            {{ $t("modal.fileSend.addMore") }}
          </v-btn>
        </div>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="show = false">{{
          $t("common.cancel")
        }}</v-btn>
        <v-btn
          color="primary"
          @click="handleSendFiles"
          :disabled="props.files.length === 0"
        >
          {{ $t("modal.fileSend.sendSrc") }}
          {{ props.files.length > 0 ? `(${props.files.length})` : "" }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.drop-zone {
  border: 2px solid transparent;
  border-radius: 12px;
  background-color: rgba(var(--v-theme-on-surface), 0.04);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.drop-zone:hover {
  background-color: rgba(var(--v-theme-primary), 0.08);
}

.drop-zone.file-drop-target-active {
  border-color: rgb(var(--v-theme-primary));
  background-color: rgba(var(--v-theme-primary), 0.12);
  transform: scale(1.01);
  box-shadow: 0 4px 12px rgba(var(--v-theme-primary), 0.15);
}

#drop-zone-list {
  transition: all 0.3s ease;
}

#drop-zone-list.file-drop-target-active {
  box-shadow: inset 0 0 0 2px rgb(var(--v-theme-primary));
  background-color: rgba(var(--v-theme-primary), 0.04);
}
</style>
