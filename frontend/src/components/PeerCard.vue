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
} from "@fortawesome/free-solid-svg-icons";
import { Peer } from "../../bindings/mesh-drop/internal/discovery";

const props = defineProps<{
  peer: Peer;
}>();

const emit = defineEmits<{
  (e: "sendFile", ip: string): void;
  (e: "sendFolder", ip: string): void;
  (e: "sendText", ip: string): void;
  (e: "sendClipboard", ip: string): void;
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
    label: "Send File",
    key: "file",
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
    case "file":
      emit("sendFile", selectedIp.value);
      break;
    case "folder":
      emit("sendFolder", selectedIp.value);
      break;
    case "text":
      emit("sendText", selectedIp.value);
      break;
    case "clipboard":
      emit("sendClipboard", selectedIp.value);
      break;
  }
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
</template>

<style scoped></style>
