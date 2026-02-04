<script setup lang="ts">
import { computed, h } from "vue";
import {
  NCard,
  NButton,
  NIcon,
  NProgress,
  NSpace,
  NText,
  NTag,
  useMessage,
  NInput,
} from "naive-ui";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
  faArrowUp,
  faArrowDown,
  faCircleExclamation,
  faUser,
  faFile,
  faFileLines,
  faFolder,
} from "@fortawesome/free-solid-svg-icons";

import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import {
  ResolvePendingRequest,
  CancelTransfer,
} from "../../bindings/mesh-drop/internal/transfer/service";
import { Dialogs, Clipboard } from "@wailsio/runtime";

import { useDialog } from "naive-ui";

const props = defineProps<{
  transfer: Transfer;
}>();

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

const percentage = computed(() =>
  Math.min(
    100,
    Math.round(
      (props.transfer.progress.current / props.transfer.progress.total) * 100,
    ),
  ),
);
const progressStatus = computed(() => {
  if (props.transfer.status === "error") return "error";
  if (props.transfer.status === "completed") return "success";
  return "default";
});

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

const message = useMessage();

const handleCopy = async () => {
  Clipboard.SetText(props.transfer.text)
    .then(() => {
      message.success("Copied to clipboard");
    })
    .catch(() => {
      message.error("Failed to copy to clipboard");
    });
};

const dialog = useDialog();
const handleOpen = async () => {
  const d = dialog.create({
    title: "Text Content",
    content: () =>
      h(NInput, {
        value: props.transfer.text,
        readonly: true,
        type: "textarea",
        rows: 10,
      }),
  });
};

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
</script>

<template>
  <n-card size="small" class="transfer-item">
    <div class="transfer-row">
      <!-- 图标 -->
      <div class="icon-wrapper">
        <n-icon size="24" v-if="props.transfer.type === 'send'" color="#38bdf8">
          <FontAwesomeIcon :icon="faArrowUp" />
        </n-icon>
        <n-icon
          size="24"
          v-else-if="props.transfer.type === 'receive'"
          color="#22c55e">
          <FontAwesomeIcon :icon="faArrowDown" />
        </n-icon>
        <n-icon size="24" v-else color="#f59e0b">
          <FontAwesomeIcon :icon="faCircleExclamation" />
        </n-icon>
      </div>

      <!-- 信息 -->
      <div class="info-wrapper">
        <div class="header-line">
          <n-text
            v-if="props.transfer.content_type === 'file'"
            strong
            class="filename"
            :title="props.transfer.file_name">
            <n-icon>
              <FontAwesomeIcon :icon="faFile" />
            </n-icon>
            {{ props.transfer.file_name }}
          </n-text>
          <n-text
            v-else-if="props.transfer.content_type === 'text'"
            strong
            class="filename"
            title="Text">
            <n-icon> <FontAwesomeIcon :icon="faFileLines" /> </n-icon>
            Text</n-text
          >
          <n-text
            v-else-if="props.transfer.content_type === 'folder'"
            strong
            class="filename"
            title="Folder">
            <n-icon> <FontAwesomeIcon :icon="faFolder" /> </n-icon>
            {{ props.transfer.file_name || "Folder" }}</n-text
          >
          <n-tag
            size="small"
            :bordered="false"
            v-if="props.transfer.sender.name">
            <template #icon>
              <n-icon>
                <FontAwesomeIcon :icon="faUser" />
              </n-icon>
            </template>
            {{ props.transfer.sender.name }}
          </n-tag>
        </div>

        <div class="meta-line">
          <n-text depth="3" class="size">{{
            formatSize(props.transfer.file_size)
          }}</n-text>

          <!-- 状态文本（进行中/已完成） -->
          <span>
            <n-text depth="3" v-if="props.transfer.status === 'active'">
              &nbsp;- {{ formatSpeed(props.transfer.progress.speed) }}</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'completed'"
              type="success">
              &nbsp;- Completed</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'error'"
              type="error">
              &nbsp;- {{ props.transfer.error_msg || "Error" }}</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'canceled'"
              type="error">
              &nbsp;- Canceled</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'rejected'"
              type="error">
              &nbsp;- Rejected</n-text
            >
          </span>
        </div>

        <!-- 文字内容 -->
        <n-text
          v-if="
            props.transfer.type === 'send' &&
            props.transfer.status === 'pending'
          "
          depth="3"
          >Waiting for accept</n-text
        >

        <!-- 进度条 -->
        <n-progress
          v-if="props.transfer.status === 'active'"
          type="line"
          :percentage="percentage"
          :status="progressStatus"
          :height="4"
          :show-indicator="false"
          processing
          style="margin-top: 4px" />
      </div>

      <!-- 接受/拒绝操作按钮 -->
      <div
        class="actions-wrapper"
        v-if="
          props.transfer.type === 'receive' &&
          props.transfer.status === 'pending'
        ">
        <n-space>
          <n-button size="small" type="success" @click="acceptTransfer">
            Accept
          </n-button>
          <n-button
            v-if="props.transfer.content_type !== 'text'"
            size="small"
            type="success"
            @click="acceptToFolder">
            Accept To Folder
          </n-button>
          <n-button size="small" type="error" ghost @click="rejectTransfer">
            Reject
          </n-button>
        </n-space>
      </div>

      <!-- 文本传输按钮 -->
      <div
        class="actions-wrapper"
        v-if="
          props.transfer.type === 'receive' &&
          props.transfer.status === 'completed' &&
          props.transfer.content_type === 'text'
        ">
        <n-space>
          <n-button size="small" type="success" @click="handleOpen"
            >Open</n-button
          >
          <n-button size="small" type="success" @click="handleCopy"
            >Copy</n-button
          >
        </n-space>
      </div>

      <!-- 取消按钮 -->
      <div class="actions-wrapper" v-if="canCancel">
        <n-space>
          <n-button
            size="small"
            type="error"
            ghost
            @click="CancelTransfer(props.transfer.id)"
            >Cancel</n-button
          >
        </n-space>
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.transfer-item {
  margin-bottom: 0.5rem;
}

.transfer-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-wrapper {
  display: flex;
  align-items: center;
}

.info-wrapper {
  flex: 1;
  min-width: 0;
}

.header-line {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.filename {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
}

.meta-line {
  font-size: 12px;
  display: flex;
  align-items: center;
}
</style>
