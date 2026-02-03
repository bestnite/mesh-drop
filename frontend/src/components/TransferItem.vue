<script setup lang="ts">
import { computed } from "vue";
import {
  NCard,
  NButton,
  NIcon,
  NProgress,
  NSpace,
  NText,
  NTag,
  useMessage,
} from "naive-ui";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
  faArrowUp,
  faArrowDown,
  faCircleExclamation,
  faUser,
} from "@fortawesome/free-solid-svg-icons";

import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import { ResolvePendingRequest } from "../../bindings/mesh-drop/internal/transfer/service";
import { Dialogs, Clipboard } from "@wailsio/runtime";

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
            :title="props.transfer.file_name"
            >{{ props.transfer.file_name }}</n-text
          >
          <n-text
            v-else-if="props.transfer.content_type === 'text'"
            strong
            class="filename"
            title="Text"
            >Text</n-text
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
              - {{ formatSpeed(props.transfer.progress.speed) }}</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'completed'"
              type="success">
              - Completed</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'error'"
              type="error">
              - {{ props.transfer.error_msg || "Error" }}</n-text
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

      <!-- 复制按钮 -->
      <div
        class="actions-wrapper"
        v-if="
          props.transfer.type === 'receive' &&
          props.transfer.status === 'completed' &&
          props.transfer.content_type === 'text'
        ">
        <n-space>
          <n-button size="small" type="success" @click="handleCopy"
            >Copy</n-button
          >
        </n-space>
      </div>

      <!-- 发送方取消按钮 -->
      <div
        class="actions-wrapper"
        v-if="
          props.transfer.type === 'send' &&
          props.transfer.status !== 'completed'
        ">
        <n-space>
          <n-button size="small" type="error" ghost @click="">
            Cancel
          </n-button>
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
