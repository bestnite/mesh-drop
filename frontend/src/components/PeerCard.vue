<script setup lang="ts">
// --- Vue 核心 ---
import { computed, ref, watch } from "vue";

// --- 组件 ---
import FileSendModal from "./modals/FileSendModal.vue";
import TextSendModal from "./modals/TextSendModal.vue";

// --- Wails & 后端绑定 ---
import { Dialogs, Clipboard } from "@wailsio/runtime";
import {
  SendFolder,
  SendText,
} from "../../bindings/mesh-drop/internal/transfer/service";
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";

// --- 属性 & 事件 ---
const props = defineProps<{
  peer: Peer;
}>();

const emit = defineEmits<{
  (e: "transferStarted"): void;
}>();

// --- 状态 ---
const selectedIp = ref<string>("");
const showFileModal = ref(false);
const showTextModal = ref(false);

const sendOptions = [
  {
    title: "Send Files",
    value: "files",
    icon: "mdi-file",
  },
  {
    title: "Send Folder",
    value: "folder",
    icon: "mdi-folder",
  },
  {
    title: "Send Text",
    value: "text",
    icon: "mdi-format-font",
  },
  {
    title: "Send Clipboard",
    value: "clipboard",
    icon: "mdi-clipboard",
  },
];

// --- 计算属性 ---
const ips = computed(() => {
  if (!props.peer.routes) return [];
  return Object.keys(props.peer.routes);
});

const osIcon = computed(() => {
  switch (props.peer.os) {
    case "linux":
      return "mdi-linux";
    case "windows":
      return "mdi-microsoft-windows";
    case "darwin":
      return "mdi-apple";
    default:
      return "mdi-desktop-classic";
  }
});

// --- 监听 ---
watch(
  ips,
  (newIps) => {
    if (newIps.length > 0) {
      if (!selectedIp.value || !newIps.includes(selectedIp.value)) {
        selectedIp.value = newIps[0];
      }
    } else {
      selectedIp.value = "";
    }
  },
  { immediate: true },
);

// --- 方法 ---
const handleAction = (key: string) => {
  if (!selectedIp.value) return;

  switch (key) {
    case "files":
      showFileModal.value = true;
      break;
    case "folder":
      handleSendFolder();
      break;
    case "text":
      showTextModal.value = true;
      break;
    case "clipboard":
      handleSendClipboard();
      break;
  }
};

const handleSendFolder = async () => {
  if (!selectedIp.value) return;
  const opts: Dialogs.OpenFileDialogOptions = {
    Title: "Select folder to send",
    CanChooseDirectories: true,
    CanChooseFiles: false,
    AllowsMultipleSelection: false,
  };
  const folderPath = await Dialogs.OpenFile(opts);
  if (!folderPath) return;

  SendFolder(props.peer, selectedIp.value, folderPath as string).catch((e) => {
    console.error(e);
    alert("Failed to send folder: " + e);
  });
  emit("transferStarted");
};

const handleSendClipboard = async () => {
  if (!selectedIp.value) return;
  const text = await Clipboard.Text();
  if (!text) {
    alert("Clipboard is empty");
    return;
  }
  SendText(props.peer, selectedIp.value, text).catch((e) => {
    console.error(e);
    alert("Failed to send clipboard: " + e);
  });
  emit("transferStarted");
};
</script>

<template>
  <v-card hover link class="peer-card pa-2">
    <template #title>
      <div class="d-flex align-center">
        <v-icon :icon="osIcon" size="24" class="mr-2"></v-icon>
        <span class="text-subtitle-1 font-weight-bold">{{ peer.name }}</span>
      </div>
    </template>

    <template #text>
      <div class="d-flex align-center flex-wrap ga-2 mt-2">
        <v-icon icon="mdi-web" size="20" class="text-medium-emphasis"></v-icon>

        <!-- Single IP Display -->
        <v-chip v-if="ips.length === 1" size="small" color="info" label>
          {{ ips[0] }}
        </v-chip>

        <!-- Multiple IP Selector -->
        <v-menu v-else-if="ips.length > 1">
          <template #activator="{ props }">
            <v-chip
              v-bind="props"
              size="small"
              color="info"
              label
              link
              append-icon="mdi-menu-down"
            >
              {{ selectedIp }}
            </v-chip>
          </template>
          <v-list density="compact">
            <v-list-item
              v-for="ip in ips"
              :key="ip"
              :value="ip"
              @click="selectedIp = ip"
            >
              <v-list-item-title>{{ ip }}</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

        <!-- No Route -->
        <v-chip v-else color="warning" size="small" label> No Route </v-chip>
      </div>
    </template>

    <template #actions>
      <v-menu>
        <template #activator="{ props }">
          <v-btn
            v-bind="props"
            block
            color="primary"
            variant="tonal"
            :disabled="ips.length === 0"
            append-icon="mdi-chevron-down"
          >
            <template #prepend>
              <v-icon icon="mdi-send"></v-icon>
            </template>
            Send
          </v-btn>
        </template>
        <v-list>
          <v-list-item
            v-for="(item, index) in sendOptions"
            :key="index"
            :value="item.value"
            @click="handleAction(item.value)"
          >
            <template #prepend>
              <v-icon :icon="item.icon"></v-icon>
            </template>
            <v-list-item-title>{{ item.title }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </template>
  </v-card>

  <!-- Modals -->
  <FileSendModal
    v-model="showFileModal"
    :peer="peer"
    :selectedIp="selectedIp"
    @transferStarted="emit('transferStarted')"
  />

  <TextSendModal
    v-model="showTextModal"
    :peer="peer"
    :selectedIp="selectedIp"
    @transferStarted="emit('transferStarted')"
  />
</template>
