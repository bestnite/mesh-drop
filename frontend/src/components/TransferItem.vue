<script setup lang="ts">
// --- Vue 核心 ---
import { computed, ref } from "vue";

// --- Wails & 后端绑定 ---
import { Dialogs, Clipboard } from "@wailsio/runtime";
import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import {
  ResolvePendingRequest,
  CancelTransfer,
  DeleteTransfer,
} from "../../bindings/mesh-drop/internal/transfer/service";

// --- 属性 & 事件 ---
const props = defineProps<{
  transfer: Transfer;
}>();

// --- 状态 ---
const showContentDialog = ref(false);

// --- 计算属性 ---
const percentage = computed(() =>
  Math.min(
    100,
    Math.round(
      (props.transfer.progress.current / props.transfer.progress.total) * 100,
    ),
  ),
);

const progressColor = computed(() => {
  if (props.transfer.status === "error") return "error";
  if (props.transfer.status === "completed") return "success";
  return "primary";
});

const canCancel = computed(() => {
  if (
    props.transfer.status === "completed" ||
    props.transfer.status === "error" ||
    props.transfer.status === "canceled" ||
    props.transfer.status === "rejected"
  ) {
    return false;
  }
  if (props.transfer.type === "send") {
    return true;
  } else if (props.transfer.type === "receive") {
    // 接收端在 pending 状态只能拒绝不能取消
    if (props.transfer.status === "pending") {
      return false;
    }
    return true;
  }
  return false;
});

const canCopy = computed(() => {
  if (
    props.transfer.type === "receive" &&
    props.transfer.status === "completed" &&
    props.transfer.content_type === "text"
  ) {
    return true;
  }
  return false;
});

const canAccept = computed(() => {
  if (
    props.transfer.type === "receive" &&
    props.transfer.status === "pending"
  ) {
    return true;
  }
  return false;
});

// --- 方法 ---
const formatSize = (bytes?: number) => {
  if (bytes === undefined) return "";
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
};

const formatSpeed = (speed?: number) => {
  if (!speed) return "";
  return formatSize(speed) + "/s";
};

const formatTime = (time: number): string => {
  return new Date(time).toLocaleString();
};

const acceptTransfer = () => {
  ResolvePendingRequest(props.transfer.id, true, "");
};

const rejectTransfer = () => {
  ResolvePendingRequest(props.transfer.id, false, "");
};

const acceptToFolder = async () => {
  const opts: Dialogs.OpenFileDialogOptions = {
    Title: "Select Folder to save the file",
    CanChooseDirectories: true,
    CanChooseFiles: false,
    AllowsMultipleSelection: false,
  };
  const path = await Dialogs.OpenFile(opts);
  if (path !== "") {
    ResolvePendingRequest(props.transfer.id, true, path as string);
  }
};

const handleDelete = () => {
  DeleteTransfer(props.transfer.id);
};

const handleCopy = async () => {
  Clipboard.SetText(props.transfer.text).catch(() => {
    console.error("Failed to copy");
  });
};
</script>

<template>
  <v-card class="transfer-item mb-2">
    <v-card-text class="py-2 px-3">
      <div class="d-flex align-center flex-wrap ga-2">
        <!-- 图标 -->
        <div>
          <v-icon
            size="24"
            v-if="props.transfer.type === 'send'"
            color="info"
            icon="mdi-arrow-up"
          ></v-icon>
          <v-icon
            size="24"
            v-else-if="props.transfer.type === 'receive'"
            color="success"
            icon="mdi-arrow-down"
          ></v-icon>
          <v-icon
            size="24"
            v-else
            color="warning"
            icon="mdi-alert-circle"
          ></v-icon>
        </div>

        <!-- 信息 -->
        <div class="info-wrapper flex-grow-1" style="min-width: 0">
          <div class="d-flex align-center ga-2 mb-1 flex-wrap">
            <div class="font-weight-bold text-truncate d-flex align-center">
              <v-icon
                size="small"
                class="mr-1"
                v-if="props.transfer.content_type === 'file'"
                icon="mdi-file"
              ></v-icon>
              <v-icon
                size="small"
                class="mr-1"
                v-else-if="props.transfer.content_type === 'text'"
                icon="mdi-file-document"
              ></v-icon>
              <v-icon
                size="small"
                class="mr-1"
                v-else-if="props.transfer.content_type === 'folder'"
                icon="mdi-folder"
              ></v-icon>
              {{
                props.transfer.file_name ||
                (props.transfer.content_type === "text" ? "Text" : "Folder")
              }}
            </div>

            <v-chip
              size="x-small"
              v-if="
                props.transfer.sender.name && props.transfer.type === 'receive'
              "
              prepend-icon="mdi-account"
            >
              {{ props.transfer.sender.name }}
            </v-chip>

            <v-chip
              size="x-small"
              v-if="props.transfer.create_time"
              prepend-icon="mdi-clock-outline"
            >
              {{ formatTime(props.transfer.create_time) }}
            </v-chip>
          </div>

          <div class="text-caption text-medium-emphasis d-flex align-center">
            <span>{{ formatSize(props.transfer.file_size) }}</span>

            <!-- 状态文本 -->
            <span v-if="props.transfer.status === 'active'">
              &nbsp;- {{ formatSpeed(props.transfer.progress.speed) }}
            </span>
            <span
              v-if="props.transfer.status === 'completed'"
              class="text-success"
            >
              &nbsp;- Completed
            </span>
            <span v-if="props.transfer.status === 'error'" class="text-error">
              &nbsp;- {{ props.transfer.error_msg || "Error" }}
            </span>
            <span v-if="props.transfer.status === 'canceled'" class="text-info">
              &nbsp;- Canceled
            </span>
            <span
              v-if="props.transfer.status === 'rejected'"
              class="text-error"
            >
              &nbsp;- Rejected
            </span>
            <span
              v-if="props.transfer.status === 'pending'"
              class="text-warning"
            >
              &nbsp;- Waiting for accept
            </span>
          </div>

          <!-- 进度条 -->
          <v-progress-linear
            v-if="props.transfer.status === 'active'"
            :model-value="percentage"
            :color="progressColor"
            height="4"
            striped
            class="mt-1"
          ></v-progress-linear>
        </div>

        <!-- 操作按钮 -->
        <div class="actions-wrapper">
          <v-btn-group density="compact" variant="tonal" divided rounded="xl">
            <v-btn v-if="canAccept" color="success" @click="acceptTransfer">
              <v-icon icon="mdi-content-save"></v-icon>
              <v-tooltip activator="parent" location="bottom">Accept</v-tooltip>
            </v-btn>

            <v-btn v-if="canAccept" color="success" @click="acceptToFolder">
              <v-icon icon="mdi-folder-arrow-right"></v-icon>
              <v-tooltip activator="parent" location="bottom">
                Save to Folder
              </v-tooltip>
            </v-btn>

            <v-btn v-if="canAccept" color="error" @click="rejectTransfer">
              <v-icon icon="mdi-close"></v-icon>
              <v-tooltip activator="parent" location="bottom">Reject</v-tooltip>
            </v-btn>

            <v-btn
              v-if="canCopy"
              color="success"
              @click="showContentDialog = true"
            >
              <v-icon icon="mdi-eye"></v-icon>
              <v-tooltip activator="parent" location="bottom">
                View Content
              </v-tooltip>
            </v-btn>

            <v-btn v-if="canCopy" color="success" @click="handleCopy">
              <v-icon icon="mdi-content-copy"></v-icon>
              <v-tooltip activator="parent" location="bottom">Copy</v-tooltip>
            </v-btn>

            <v-btn
              v-if="
                props.transfer.status === 'completed' ||
                props.transfer.status === 'error' ||
                props.transfer.status === 'canceled' ||
                props.transfer.status === 'rejected'
              "
              color="info"
              @click="handleDelete"
            >
              <v-icon icon="mdi-delete"></v-icon>
              <v-tooltip activator="parent" location="bottom">Delete</v-tooltip>
            </v-btn>

            <v-btn
              v-if="canCancel"
              color="error"
              @click="CancelTransfer(props.transfer.id)"
            >
              <v-icon icon="mdi-stop"></v-icon>
              <v-tooltip activator="parent" location="bottom">Cancel</v-tooltip>
            </v-btn>
          </v-btn-group>
        </div>
      </div>
    </v-card-text>
  </v-card>

  <v-dialog v-model="showContentDialog" width="600">
    <v-card title="Text Content">
      <v-card-text>
        <v-textarea
          :model-value="props.transfer.text"
          readonly
          rows="10"
        ></v-textarea>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.info-wrapper {
  overflow: hidden;
}
</style>
