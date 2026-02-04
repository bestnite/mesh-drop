<script setup lang="ts">
// --- Vue 核心 ---
import { computed, ref, watch } from "vue";

// --- Wails & 后端绑定 ---
import { Events, Dialogs } from "@wailsio/runtime";
import { SendFiles } from "../../../bindings/mesh-drop/internal/transfer/service";
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
const fileList = ref<{ name: string; path: string }[]>([]);

// --- 计算属性 ---
const show = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

// --- 监听 ---
watch(show, (newVal) => {
  if (newVal) {
    Events.On("files-dropped", (event) => {
      const files: string[] = event.data.files || [];
      files.forEach((f) => {
        if (!fileList.value.find((existing) => existing.path === f)) {
          fileList.value.push({
            name: f.split(/[\/]/).pop() || f,
            path: f,
          });
        }
      });
    });
  } else {
    Events.Off("files-dropped");
    fileList.value = [];
  }
});

// --- 方法 ---
const openFileDialog = async () => {
  const files = await Dialogs.OpenFile({
    Title: "Select files to send",
    AllowsMultipleSelection: true,
  });

  if (files) {
    if (Array.isArray(files)) {
      files.forEach((f) => {
        if (!fileList.value.find((existing) => existing.path === f)) {
          fileList.value.push({
            name: f.split(/[\\/]/).pop() || f,
            path: f,
          });
        }
      });
    } else {
      const f = files as string;
      if (!fileList.value.find((existing) => existing.path === f)) {
        fileList.value.push({
          name: f.split(/[\\/]/).pop() || f,
          path: f,
        });
      }
    }
  }
};

const handleRemoveFile = (index: number) => {
  fileList.value.splice(index, 1);
};

const handleSendFiles = async () => {
  if (fileList.value.length === 0 || !props.selectedIp) return;
  const paths = fileList.value.map((f) => f.path);

  try {
    await SendFiles(props.peer, props.selectedIp, paths);
    emit("transferStarted");
    show.value = false;
  } catch (e) {
    console.error(e);
    alert("Failed to send files: " + e);
  }
};
</script>

<template>
  <v-dialog v-model="show" width="600" persistent eager>
    <v-card title="Send Files">
      <v-card-text>
        <div
          v-if="fileList.length === 0"
          class="drop-zone pa-10 text-center rounded-lg border-dashed"
          @click="openFileDialog"
          data-file-drop-target
        >
          <v-icon
            icon="mdi-cloud-upload"
            size="48"
            color="primary"
            class="mb-2"
          ></v-icon>
          <div class="text-body-1 text-medium-emphasis">
            Click to select files
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
          >
            <v-list-item
              v-for="(file, index) in fileList"
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
            variant="outlined"
            style="border-style: dashed"
            prepend-icon="mdi-plus"
            @click="openFileDialog"
            class="mt-2"
          >
            Add more files
          </v-btn>
        </div>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="show = false">Cancel</v-btn>
        <v-btn
          color="primary"
          @click="handleSendFiles"
          :disabled="fileList.length === 0"
        >
          Send {{ fileList.length > 0 ? `(${fileList.length})` : "" }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.drop-zone {
  border: 2px dashed #666; /* Use a darker color or theme var */
  cursor: pointer;
  transition: all 0.3s;
}
.drop-zone:hover {
  border-color: #38bdf8;
  background-color: rgba(56, 189, 248, 0.05);
}

.drop-zone.file-drop-target-active {
  border-color: #38bdf8;
  background-color: rgba(56, 189, 248, 0.1);
}
</style>
