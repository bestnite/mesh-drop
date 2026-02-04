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
  NDropdown,
  NButtonGroup,
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
  faClock,
  faChevronDown,
  faEye,
  faCopy,
  faTrash,
  faXmark,
  faStop,
  faCheck,
} from "@fortawesome/free-solid-svg-icons";

import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import {
  ResolvePendingRequest,
  CancelTransfer,
  DeleteTransfer,
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

const formatTime = (time: number): string => {
  return new Date(time).toLocaleString();
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

const dropdownOptions = [
  {
    label: "Accept To Folder",
    key: "folder",
  },
];

const handleSelect = (key: string | number) => {
  if (key === "folder") {
    acceptToFolder();
  }
};

const handleDelete = () => {
  DeleteTransfer(props.transfer.id);
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
            v-if="
              props.transfer.sender.name && props.transfer.type === 'receive'
            ">
            <template #icon>
              <n-icon>
                <FontAwesomeIcon :icon="faUser" />
              </n-icon>
            </template>
            {{ props.transfer.sender.name }}
          </n-tag>
          <n-tag
            size="small"
            :bordered="false"
            v-if="props.transfer.create_time">
            <template #icon>
              <n-icon>
                <FontAwesomeIcon :icon="faClock" />
              </n-icon>
            </template>
            {{ formatTime(props.transfer.create_time) }}
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
              type="info">
              &nbsp;- Canceled</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'rejected'"
              type="error">
              &nbsp;- Rejected</n-text
            >
            <n-text
              depth="3"
              v-if="props.transfer.status === 'pending'"
              type="warning">
              &nbsp;- Waiting for accept</n-text
            >
          </span>
        </div>

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

      <!-- 操作按钮 -->
      <div class="actions-wrapper">
        <n-space>
          <n-button-group size="small">
            <n-button v-if="canAccept" type="success" @click="acceptTransfer">
              <template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faCheck" />
                </n-icon>
              </template>
            </n-button>
            <n-dropdown
              trigger="click"
              :options="dropdownOptions"
              @select="handleSelect"
              v-if="canAccept && props.transfer.content_type !== 'text'">
              <n-button type="success">
                <template #icon>
                  <n-icon>
                    <FontAwesomeIcon :icon="faChevronDown" />
                  </n-icon>
                </template>
              </n-button>
            </n-dropdown>
            <n-button
              v-if="canAccept"
              size="small"
              type="error"
              @click="rejectTransfer">
              <template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faXmark" />
                </n-icon>
              </template>
            </n-button>

            <n-button type="success" @click="handleOpen" v-if="canCopy"
              ><template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faEye" />
                </n-icon>
              </template>
            </n-button>
            <n-button type="success" @click="handleCopy" v-if="canCopy"
              ><template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faCopy" />
                </n-icon>
              </template>
            </n-button>
            <n-button
              type="success"
              @click="handleDelete"
              v-if="
                props.transfer.status === 'completed' ||
                props.transfer.status === 'error' ||
                props.transfer.status === 'canceled' ||
                props.transfer.status === 'rejected'
              ">
              <template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faTrash" />
                </n-icon>
              </template>
            </n-button>
            <n-button
              v-if="canCancel"
              size="small"
              type="error"
              @click="CancelTransfer(props.transfer.id)"
              ><template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faStop" />
                </n-icon>
              </template>
            </n-button>
          </n-button-group>
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
  flex-wrap: wrap;
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

@media (max-width: 640px) {
  .actions-wrapper {
    width: 100%;
    margin-top: 8px;
    display: flex;
    justify-content: flex-end;
  }

  .transfer-row {
    gap: 8px;
  }
}
</style>
