<script setup lang="ts">
import { computed, ref, watch } from "vue";
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
        <div v-else-if="ips.length > 1" style="width: 150px">
          <v-select
            v-model="selectedIp"
            :items="ips"
            density="compact"
            hide-details
            variant="outlined"
            single-line
          ></v-select>
        </div>

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

  <!-- 文件发送 Modal -->
  <v-dialog v-model="showFileModal" width="600" persistent eager>
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
        <v-btn variant="text" @click="handleCancelFiles">Cancel</v-btn>
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

  <!-- 文本发送 Modal -->
  <v-dialog v-model="showTextModal" width="500" persistent eager>
    <v-card title="Send Text">
      <v-card-text>
        <v-textarea
          v-model="textContent"
          label="Content"
          placeholder="Type something to send..."
          rows="4"
          auto-grow
        ></v-textarea>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="text" @click="showTextModal = false">Cancel</v-btn>
        <v-btn
          color="primary"
          @click="executeSendText"
          :disabled="!textContent"
        >
          Send
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
