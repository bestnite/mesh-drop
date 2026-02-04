<script setup lang="ts">
import { computed, ref, watch, h } from "vue";
import {
  NCard,
  NButton,
  NIcon,
  NTag,
  NSpace,
  NDropdown,
  NSelect,
  type DropdownOption,
  NModal,
  NList,
  NListItem,
  NThing,
  NEmpty,
  NInput,
} from "naive-ui";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
  faLinux,
  faWindows,
  faApple,
} from "@fortawesome/free-brands-svg-icons";
import {
  faDesktop,
  faGlobe,
  faPaperPlane,
  faChevronDown,
  faFile,
  faFolder,
  faFont,
  faClipboard,
  faTrash,
  faPlus,
  faCloudArrowUp,
} from "@fortawesome/free-solid-svg-icons";
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";
import { Dialogs, Events, Clipboard } from "@wailsio/runtime";
import {
  SendFiles,
  SendFolder,
  SendText,
} from "../../bindings/mesh-drop/internal/transfer/service";

const props = defineProps<{
  peer: Peer;
}>();

const emit = defineEmits<{
  (e: "transferStarted"): void;
}>();

const ips = computed(() => {
  if (!props.peer.routes) return [];
  return Object.keys(props.peer.routes);
});

const selectedIp = ref<string>("");

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

const ipOptions = computed(() => {
  return ips.value.map((ip) => ({
    label: ip,
    value: ip,
  }));
});

const osIcon = computed(() => {
  switch (props.peer.os) {
    case "linux":
      return faLinux;
    case "windows":
      return faWindows;
    case "darwin":
      return faApple;
    default:
      return faDesktop;
  }
});

const sendOptions: DropdownOption[] = [
  {
    label: "Send Files",
    key: "files",
    icon: () =>
      h(NIcon, null, { default: () => h(FontAwesomeIcon, { icon: faFile }) }),
  },
  {
    label: "Send Folder",
    key: "folder",
    icon: () =>
      h(NIcon, null, { default: () => h(FontAwesomeIcon, { icon: faFolder }) }),
  },
  {
    label: "Send Text",
    key: "text",
    icon: () =>
      h(NIcon, null, { default: () => h(FontAwesomeIcon, { icon: faFont }) }),
  },
  {
    label: "Send Clipboard",
    key: "clipboard",
    icon: () =>
      h(NIcon, null, {
        default: () => h(FontAwesomeIcon, { icon: faClipboard }),
      }),
  },
];

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

// --- 发送逻辑 ---

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

// --- 文本发送 ---
const showTextModal = ref(false);
const textContent = ref("");

const executeSendText = async () => {
  if (!selectedIp.value || !textContent.value) return;
  SendText(props.peer, selectedIp.value, textContent.value).catch((e) => {
    console.error(e);
    alert("Failed to send text: " + e);
  });
  emit("transferStarted");
  showTextModal.value = false;
  textContent.value = "";
};

// --- 文件选择 ---
const showFileModal = ref(false);
watch(showFileModal, (newVal) => {
  if (newVal) {
    Events.On("files-dropped", (event) => {
      fileList.value = event.data.files.map((f) => ({
        name: f.split(/[\/]/).pop() || f,
        path: f,
      }));
    });
  } else {
    Events.Off("files-dropped");
  }
});
const fileList = ref<{ name: string; path: string }[]>([]);

const openFileDialog = async () => {
  const files = await Dialogs.OpenFile({
    Title: "Select files to send",
    AllowsMultipleSelection: true,
  });
  if (files) {
    if (Array.isArray(files)) {
      files.forEach((f) => {
        // 去重
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

const handleCancelFiles = () => {
  showFileModal.value = false;
  fileList.value = [];
};

const handleSendFiles = () => {
  if (fileList.value.length === 0 || !selectedIp.value) return;
  const paths = fileList.value.map((f) => f.path);
  SendFiles(props.peer, selectedIp.value, paths).catch((e) => {
    console.error(e);
    alert("Failed to send files: " + e);
  });
  emit("transferStarted");
  handleCancelFiles();
};
</script>

<template>
  <n-card hoverable class="peer-card">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <n-icon size="24">
          <FontAwesomeIcon :icon="osIcon" />
        </n-icon>
        <span style="user-select: none">{{ peer.name }}</span>
      </div>
    </template>

    <n-space vertical>
      <div style="display: flex; align-items: center; gap: 8px">
        <n-icon>
          <FontAwesomeIcon :icon="faGlobe" />
        </n-icon>
        <!-- Single IP Display -->
        <n-tag
          v-if="ips.length === 1"
          :bordered="false"
          type="info"
          size="small">
          {{ ips[0] }}
        </n-tag>
        <!-- Multiple IP Selector -->
        <n-select
          v-else-if="ips.length > 1"
          v-model:value="selectedIp"
          :options="ipOptions"
          size="small"
          style="width: 140px" />
        <!-- No Route -->
        <n-tag v-else :bordered="false" type="warning" size="small">
          No Route
        </n-tag>
      </div>
    </n-space>

    <template #action>
      <div style="display: flex; gap: 8px">
        <n-dropdown
          trigger="click"
          :options="sendOptions"
          @select="handleAction"
          :disabled="ips.length === 0">
          <n-button type="primary" block dashed style="width: 100%">
            <template #icon>
              <n-icon>
                <FontAwesomeIcon :icon="faPaperPlane" />
              </n-icon>
            </template>
            Send...
            <n-icon style="margin-left: 4px">
              <FontAwesomeIcon :icon="faChevronDown" />
            </n-icon>
          </n-button>
        </n-dropdown>
      </div>
    </template>
  </n-card>

  <n-modal
    v-model:show="showFileModal"
    preset="card"
    title="Send Files"
    style="width: 600px; max-width: 90%"
    :bordered="false">
    <div
      v-if="fileList.length === 0"
      class="drop-zone"
      @click="openFileDialog"
      data-file-drop-target>
      <n-empty description="Click to select files">
        <template #icon>
          <n-icon :size="48">
            <FontAwesomeIcon :icon="faCloudArrowUp" />
          </n-icon>
        </template>
      </n-empty>
    </div>

    <div v-else>
      <div
        style="max-height: 400px; overflow-y: auto; margin-bottom: 16px"
        data-file-drop-target>
        <n-list bordered>
          <n-list-item v-for="(file, index) in fileList" :key="file.path">
            <template #suffix>
              <n-button text type="error" @click="handleRemoveFile(index)">
                <template #icon>
                  <n-icon><FontAwesomeIcon :icon="faTrash" /></n-icon>
                </template>
              </n-button>
            </template>
            <n-thing :title="file.name" :description="file.path"></n-thing>
          </n-list-item>
        </n-list>
      </div>
      <n-button dashed block @click="openFileDialog">
        <template #icon>
          <n-icon><FontAwesomeIcon :icon="faPlus" /></n-icon>
        </template>
        Add more files
      </n-button>
    </div>

    <template #footer>
      <n-space justify="end">
        <n-button @click="handleCancelFiles">Cancel</n-button>
        <n-button
          type="primary"
          @click="handleSendFiles"
          :disabled="fileList.length === 0">
          Send {{ fileList.length > 0 ? `(${fileList.length})` : "" }}
        </n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 文本发送 Modal -->
  <n-modal
    v-model:show="showTextModal"
    preset="card"
    title="Send Text"
    style="width: 500px; max-width: 90%"
    :bordered="false">
    <n-input
      v-model:value="textContent"
      type="textarea"
      placeholder="Type something to send..."
      :autosize="{ minRows: 4, maxRows: 10 }" />
    <template #footer>
      <n-space justify="end">
        <n-button @click="showTextModal = false">Cancel</n-button>
        <n-button
          type="primary"
          @click="executeSendText"
          :disabled="!textContent">
          Send
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.drop-zone {
  border: 2px dashed #ccc;
  border-radius: 8px;
  padding: 40px;
  text-align: center;
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
