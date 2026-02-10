<script setup lang="ts">
// --- Vue 核心 ---
import { computed, ref, watch, onMounted, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";

// --- 组件 ---
import FileSendModal from "./modals/FileSendModal.vue";
import TextSendModal from "./modals/TextSendModal.vue";

// --- Wails & 后端绑定 ---
import { Dialogs, Clipboard, Events } from "@wailsio/runtime";
import {
  SendFolder,
  SendText,
} from "../../bindings/mesh-drop/internal/transfer/service";
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";
import {
  IsTrusted,
  AddTrust,
  RemoveTrust,
} from "../../bindings/mesh-drop/internal/config/config";
import { File } from "bindings/mesh-drop/models";

// --- 生命周期 ---
const droppedFiles = ref<File[]>([]);
onMounted(async () => {
  try {
    isTrusted.value = await IsTrusted(props.peer.id);
  } catch (err) {
    console.error("Failed to check trusted peer status:", err);
  }
  Events.On("files-dropped", (event) => {
    droppedFiles.value = event.data.files;
    showFileModal.value = true;
  });
});

onUnmounted(() => {
  Events.Off("files-dropped");
});

// --- 属性 & 事件 ---
const props = defineProps<{
  peer: Peer;
}>();

const { t } = useI18n();

const emit = defineEmits<{
  (e: "transferStarted"): void;
}>();

// --- 状态 ---
const selectedIp = ref<string>("");
const showFileModal = ref(false);
const showTextModal = ref(false);
const isTrusted = ref(false);

const sendOptions = computed(() => [
  {
    title: t("discover.sendFiles"),
    value: "files",
    icon: "mdi-file",
  },
  {
    title: t("discover.sendFolder"),
    value: "folder",
    icon: "mdi-folder",
  },
  {
    title: t("discover.sendText"),
    value: "text",
    icon: "mdi-format-font",
  },
  {
    title: t("discover.sendClipboard"),
    value: "clipboard",
    icon: "mdi-clipboard",
  },
]);

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

const showMismatch = computed(() => {
  return props.peer.trust_mismatch && isTrusted.value;
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
    Title: t("discover.selectFolder"),
    CanChooseDirectories: true,
    CanChooseFiles: false,
    AllowsMultipleSelection: false,
  };
  const folderPath = await Dialogs.OpenFile(opts);
  if (!folderPath) return;

  SendFolder(props.peer, selectedIp.value, folderPath as string).catch((e) => {
    console.error(e);
    alert(t("discover.sendFolderFailed", { error: e }));
  });
  emit("transferStarted");
};

const handleSendClipboard = async () => {
  if (!selectedIp.value) return;
  const text = await Clipboard.Text();
  if (!text) {
    alert(t("discover.clipboardEmpty"));
    return;
  }
  SendText(props.peer, selectedIp.value, text).catch((e) => {
    console.error(e);
    alert(t("discover.sendClipboardFailed", { error: e }));
  });
  emit("transferStarted");
};

const handleTrust = () => {
  AddTrust(props.peer.id, props.peer.pk);
  isTrusted.value = true;
};

const handleUntrust = () => {
  RemoveTrust(props.peer.id);
  isTrusted.value = false;
};
</script>

<template>
  <v-card
    hover
    link
    class="peer-card pa-2"
    :ripple="false"
    data-file-drop-target
    :id="`drop-zone-peer-${peer.id}`"
  >
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
        <v-chip v-else color="warning" size="small" label>
          {{ t("discover.noRoute") }}
        </v-chip>
      </div>

      <!-- 拖放提示覆盖层 -->
      <div class="drag-drop-overlay">
        <v-icon
          icon="mdi-file-upload-outline"
          size="48"
          color="primary"
          style="opacity: 0.8"
        ></v-icon>
      </div>
    </template>

    <v-card-actions>
      <!-- Trust Mismatch Warning -->
      <v-btn
        v-if="showMismatch"
        class="flex-grow-1"
        color="warning"
        variant="tonal"
        prepend-icon="mdi-alert"
        :ripple="false"
        style="pointer-events: none; min-width: 0"
      >
        <span class="text-truncate">{{ t("discover.mismatch") }}</span>
      </v-btn>

      <v-menu v-else>
        <template #activator="{ props }">
          <v-btn
            v-bind="props"
            class="flex-grow-1"
            color="primary"
            variant="tonal"
            :disabled="ips.length === 0"
            append-icon="mdi-chevron-down"
          >
            <template #prepend>
              <v-icon icon="mdi-send"></v-icon>
            </template>
            {{ t("discover.send") }}
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

      <!-- Trust Mismatch Reset Override -->
      <v-btn
        v-if="showMismatch"
        variant="tonal"
        color="error"
        @click="handleUntrust"
      >
        <v-icon icon="mdi-delete"></v-icon>
        <v-tooltip activator="parent" location="bottom">{{
          t("discover.resetTrust")
        }}</v-tooltip>
      </v-btn>

      <v-btn
        v-else-if="!isTrusted"
        variant="tonal"
        color="primary"
        @click="handleTrust"
      >
        <v-icon icon="mdi-star-outline"></v-icon>
        <v-tooltip activator="parent" location="bottom">{{
          t("discover.trustPeer")
        }}</v-tooltip>
      </v-btn>
      <v-btn v-else variant="tonal" color="primary" @click="handleUntrust">
        <v-icon icon="mdi-star"></v-icon>
        <v-tooltip activator="parent" location="bottom">{{
          t("discover.untrustPeer")
        }}</v-tooltip>
      </v-btn>
    </v-card-actions>
  </v-card>

  <!-- Modals -->
  <FileSendModal
    v-model="showFileModal"
    :peer="peer"
    :selectedIp="selectedIp"
    :files="droppedFiles"
    @transferStarted="emit('transferStarted')"
  />

  <TextSendModal
    v-model="showTextModal"
    :peer="peer"
    :selectedIp="selectedIp"
    @transferStarted="emit('transferStarted')"
  />
</template>

<style scoped>
.peer-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.peer-card::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgb(var(--v-theme-primary));
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
  z-index: 1;
}

.peer-card.file-drop-target-active {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px -4px rgba(var(--v-theme-primary), 0.24) !important;
  border-color: rgb(var(--v-theme-primary)) !important;
}

.peer-card.file-drop-target-active::after {
  opacity: 0.12;
}

.drag-drop-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.3s ease;
  background: rgba(var(--v-theme-surface), 0.8);
  backdrop-filter: blur(2px);
}

.peer-card.file-drop-target-active .drag-drop-overlay {
  opacity: 1;
}

.drag-drop-content {
  color: rgb(var(--v-theme-primary));
  font-weight: 500;
  display: flex;
  flex-direction: column;
  align-items: center;
  transform: translateY(10px);
  transition: transform 0.3s ease;
}

.peer-card.file-drop-target-active .drag-drop-content {
  transform: translateY(0);
}
</style>
